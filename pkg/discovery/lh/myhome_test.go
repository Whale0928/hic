package lh

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMyHomeClient_ListNotices_공공임대응답을파싱한다(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rsdtRcritNtcList" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("serviceKey") != "test-key" ||
			r.URL.Query().Get("pageNo") != "1" ||
			r.URL.Query().Get("numOfRows") != "2" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"response": {
				"header": {"resultCode": "00", "resultMsg": "NORMAL SERVICE"},
				"body": {
					"totalCount": "1",
					"item": [{
						"pblancId": "20364",
						"houseSn": 1,
						"pblancNm": "함평기산 통합공공임대주택 입주자모집",
						"suplyInsttNm": "LH",
						"houseTyNm": "아파트",
						"suplyTyNm": "통합공공임대",
						"rcritPblancDe": "20260519",
						"beginDe": "20260605",
						"endDe": "20260609",
						"hsmpNm": "함평기산",
						"brtcNm": "전라남도",
						"signguNm": "함평군",
						"fullAdres": "전라남도 함평군 함평읍 기각리 775-1",
						"refrnLegaldongNm": "함평읍",
						"sumSuplyCo": 60,
						"rentGtn": 7303000,
						"mtRntchrg": 109640,
						"url": "https://apply.lh.or.kr/lhapply/apply/wt/wrtanc/selectWrtancInfo.do",
						"pcUrl": "https://www.myhome.go.kr/hws/portal/sch/selectRsdtRcritNtcDetailView.do?pblancId=20364&houseSn=1"
					}]
				}
			}
		}`))
	}))
	defer server.Close()

	client := MyHomeClient{BaseURL: server.URL, ServiceKey: "test-key", HTTPClient: server.Client()}
	page, err := client.ListNotices(context.Background(), MyHomeRental, 1, 2)
	if err != nil {
		t.Fatalf("ListNotices() error = %v", err)
	}

	if page.TotalCount != 1 || len(page.Items) != 1 {
		t.Fatalf("page = %+v", page)
	}
	item := page.Items[0]
	if item.SourceSeq() != "20364:1" {
		t.Fatalf("SourceSeq() = %q", item.SourceSeq())
	}
	if item.Title != "함평기산 통합공공임대주택 입주자모집" || item.Agency != "LH" {
		t.Fatalf("item = %+v", item)
	}
	if item.SupplyCount == nil || *item.SupplyCount != 60 {
		t.Fatalf("SupplyCount = %v", item.SupplyCount)
	}
}

func TestMyHomeClient_ListNotices_요청에러에서서비스키를마스킹한다(t *testing.T) {
	client := MyHomeClient{
		BaseURL:    "https://example.test",
		ServiceKey: "secret-key",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("request failed: %s", req.URL.String())
		})},
	}

	_, err := client.ListNotices(context.Background(), MyHomeRental, 3, 200)
	if err == nil {
		t.Fatal("ListNotices() error = nil")
	}
	if strings.Contains(err.Error(), "secret-key") {
		t.Fatalf("error leaked service key: %v", err)
	}
	if !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("error = %v, want redacted marker", err)
	}
}

func TestMyHomeClient_ListNotices_일시요청에러를재시도한다(t *testing.T) {
	attempts := 0
	client := MyHomeClient{
		BaseURL:    "https://example.test",
		ServiceKey: "test-key",
		HTTPClient: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return nil, fmt.Errorf("temporary failure")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{
					"response": {
						"header": {"resultCode": "00", "resultMsg": "NORMAL SERVICE"},
						"body": {"totalCount": 0, "item": null}
					}
				}`)),
			}, nil
		})},
	}

	_, err := client.ListNotices(context.Background(), MyHomeRental, 1, 200)
	if err != nil {
		t.Fatalf("ListNotices() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
