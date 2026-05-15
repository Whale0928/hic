package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
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

func TestServer_Display_HTML앱을제공한다(t *testing.T) {
	displayDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(displayDir, "index.html"), []byte("<html><body>HIC display</body></html>"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	e := NewWithDisplay(fakeRepository{}, displayDir)
	req := httptest.NewRequest(http.MethodGet, "/display/index.html", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "HIC display") {
		t.Fatalf("body = %q, want HIC display", rec.Body.String())
	}
}

func TestServer_PDFOfferingsReport_PDF를실제로추출해JSON으로제공한다(t *testing.T) {
	pdfPath := filepath.Join(t.TempDir(), "sample.pdf")
	if err := os.WriteFile(pdfPath, minimalPDF("Hello HIC"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	e := NewWithDisplay(fakeRepository{}, t.TempDir())
	req := httptest.NewRequest(http.MethodGet, "/reports/pdf-offerings?file="+pdfPath, nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	var body struct {
		Totals struct {
			Files     int `json:"files"`
			Artifacts int `json:"artifacts"`
			Offerings int `json:"offerings"`
		} `json:"totals"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Unmarshal() error = %v: %s", err, rec.Body.String())
	}
	if body.Totals.Files != 1 || body.Totals.Artifacts != 1 || body.Totals.Offerings != 0 {
		t.Fatalf("totals = %+v", body.Totals)
	}
}

func TestServer_PDFOfferingsReport_파일파라미터를요구한다(t *testing.T) {
	e := NewWithDisplay(fakeRepository{}, t.TempDir())
	req := httptest.NewRequest(http.MethodGet, "/reports/pdf-offerings", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
	}
}

func minimalPDF(text string) []byte {
	stream := "BT /F1 24 Tf 100 700 Td (" + text + ") Tj ET"
	objects := []string{
		`<< /Type /Catalog /Pages 2 0 R >>`,
		`<< /Type /Pages /Kids [3 0 R] /Count 1 >>`,
		`<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>`,
		`<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>`,
		"<< /Length " + strconv.Itoa(len(stream)) + " >>\nstream\n" + stream + "\nendstream",
	}

	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = b.Len()
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(" 0 obj\n")
		b.WriteString(obj)
		b.WriteString("\nendobj\n")
	}
	xrefOffset := b.Len()
	b.WriteString("xref\n0 ")
	b.WriteString(strconv.Itoa(len(objects) + 1))
	b.WriteString("\n0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		b.WriteString(leftPadInt(offsets[i], 10))
		b.WriteString(" 00000 n \n")
	}
	b.WriteString("trailer\n<< /Size ")
	b.WriteString(strconv.Itoa(len(objects) + 1))
	b.WriteString(" /Root 1 0 R >>\nstartxref\n")
	b.WriteString(strconv.Itoa(xrefOffset))
	b.WriteString("\n%%EOF\n")
	return []byte(b.String())
}

func leftPadInt(n int, width int) string {
	value := strconv.Itoa(n)
	for len(value) < width {
		value = "0" + value
	}
	return value
}
