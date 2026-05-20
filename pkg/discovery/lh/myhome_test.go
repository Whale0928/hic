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

func TestMyHomeClient_ListNotices_공공분양납부금액을파싱한다(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ltRsdtRcritNtcList" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"response": {
				"header": {"resultCode": "00", "resultMsg": "NORMAL SERVICE"},
				"body": {
					"totalCount": "1",
					"item": {
						"pblancId": "1411",
						"houseSn": 1,
						"pblancNm": "[정정공고]인천계양 A9블록 신혼희망타운(공공분양) 입주자모집공고",
						"suplyInsttNm": "LH",
						"houseTyNm": "아파트",
						"hsmpNm": "인천계양 A9블록",
						"sumSuplyCo": 317,
						"enty": 46654000,
						"prtpay": 93308000,
						"surlus": 271578000,
						"beginDe": "20260518",
						"endDe": "20260528"
					}
				}
			}
		}`))
	}))
	defer server.Close()

	client := MyHomeClient{BaseURL: server.URL, ServiceKey: "test-key", HTTPClient: server.Client()}
	page, err := client.ListNotices(context.Background(), MyHomeSale, 1, 1)
	if err != nil {
		t.Fatalf("ListNotices() error = %v", err)
	}

	item := page.Items[0]
	if item.ContractPaymentKRW == nil || *item.ContractPaymentKRW != 46654000 {
		t.Fatalf("ContractPaymentKRW = %v", item.ContractPaymentKRW)
	}
	if item.InterimPaymentKRW == nil || *item.InterimPaymentKRW != 93308000 {
		t.Fatalf("InterimPaymentKRW = %v", item.InterimPaymentKRW)
	}
	if item.BalancePaymentKRW == nil || *item.BalancePaymentKRW != 271578000 {
		t.Fatalf("BalancePaymentKRW = %v", item.BalancePaymentKRW)
	}
}

func TestMyHomeClient_ListNotices_sumSuplyCo가0이면공급호수문구에서보정한다(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"response": {
				"header": {"resultCode": "00", "resultMsg": "NORMAL SERVICE"},
				"body": {
					"totalCount": "1",
					"item": {
						"pblancId": "19631",
						"houseSn": 0,
						"pblancNm": "2026년 청년 전세임대 1순위 입주자 수시모집",
						"suplyInsttNm": "LH",
						"houseTyNm": "아파트",
						"suplyTyNm": "전세임대",
						"sumSuplyCo": 0,
						"suplyHoCo": "7,000호\r\n* 1순위 수요에 따라 목표 물량 조기 소진의 경우 1순위 추가 공급 가능",
						"rentGtn": 0,
						"mtRntchrg": 0
					}
				}
			}
		}`))
	}))
	defer server.Close()

	client := MyHomeClient{BaseURL: server.URL, ServiceKey: "test-key", HTTPClient: server.Client()}
	page, err := client.ListNotices(context.Background(), MyHomeRental, 1, 1)
	if err != nil {
		t.Fatalf("ListNotices() error = %v", err)
	}

	item := page.Items[0]
	if item.SupplyCount == nil || *item.SupplyCount != 7000 {
		t.Fatalf("SupplyCount = %v, want 7000", item.SupplyCount)
	}
	if item.DepositKRW != nil || item.MonthlyRent != nil {
		t.Fatalf("zero API money should be treated as unavailable: deposit=%v rent=%v", item.DepositKRW, item.MonthlyRent)
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
