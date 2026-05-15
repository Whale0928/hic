package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hic/pkg/persistence"
)

type fakeRepository struct{}

func (fakeRepository) ListOfferings(ctx context.Context, limit int32, qaStatus string) ([]persistence.OfferingView, error) {
	return []persistence.OfferingView{
		{ID: 1, Agency: "SH", HousingName: "청년주택", UnitNo: qaStatus, UnitType: "36A", SourceSpan: "xlsx://주택목록!2"},
	}, nil
}

func (fakeRepository) ListSourceNotices(ctx context.Context, limit int32) ([]persistence.SourceNoticeView, error) {
	return []persistence.SourceNoticeView{
		{ID: 10, Agency: "SH", BoardKind: "rental", Seq: "296598", Title: "입주자 모집공고"},
	}, nil
}

func TestServer_Health(t *testing.T) {
	e := New(fakeRepository{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("body = %v", body)
	}
}

func TestServer_Offerings(t *testing.T) {
	e := New(fakeRepository{})
	req := httptest.NewRequest(http.MethodGet, "/offerings?limit=1", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var offerings []persistence.OfferingView
	if err := json.Unmarshal(rec.Body.Bytes(), &offerings); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(offerings) != 1 || offerings[0].UnitNo != "approved" {
		t.Fatalf("offerings = %+v", offerings)
	}
}

func TestServer_Offerings_QA상태를쿼리로지정한다(t *testing.T) {
	e := New(fakeRepository{})
	req := httptest.NewRequest(http.MethodGet, "/offerings?limit=1&qa_status=pending", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var offerings []persistence.OfferingView
	if err := json.Unmarshal(rec.Body.Bytes(), &offerings); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(offerings) != 1 || offerings[0].UnitNo != "pending" {
		t.Fatalf("offerings = %+v", offerings)
	}
}

func TestServer_Resources_HTML리포트를제공한다(t *testing.T) {
	resourcesDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(resourcesDir, "pdf-offerings.html"), []byte("<html><body>PDF report</body></html>"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	e := NewWithResources(fakeRepository{}, resourcesDir)
	req := httptest.NewRequest(http.MethodGet, "/resources/pdf-offerings.html", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "PDF report") {
		t.Fatalf("body = %q, want PDF report", rec.Body.String())
	}
}

func TestServer_PDFOfferingsReport_JSON리포트를제공한다(t *testing.T) {
	resourcesDir := t.TempDir()
	report := []byte(`{"generated_at":"2026-05-15T00:00:00Z","offerings":[{"housing_name":"정릉 희망하우징"}]}`)
	if err := os.WriteFile(filepath.Join(resourcesDir, "pdf-offerings.json"), report, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	e := NewWithResources(fakeRepository{}, resourcesDir)
	req := httptest.NewRequest(http.MethodGet, "/reports/pdf-offerings", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if !strings.Contains(rec.Body.String(), "정릉 희망하우징") {
		t.Fatalf("body = %q, want report json", rec.Body.String())
	}
}
