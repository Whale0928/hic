package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hic/pkg/persistence"
)

type fakeRepository struct{}

func (fakeRepository) ListHousingUnits(ctx context.Context, limit int32) ([]persistence.HousingUnitView, error) {
	return []persistence.HousingUnitView{
		{ID: 1, Agency: "SH", HousingName: "청년주택", UnitNo: "1201", UnitType: "36A", SourceSpan: "xlsx://주택목록!2"},
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

func TestServer_Units(t *testing.T) {
	e := New(fakeRepository{})
	req := httptest.NewRequest(http.MethodGet, "/units?limit=1", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var units []persistence.HousingUnitView
	if err := json.Unmarshal(rec.Body.Bytes(), &units); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(units) != 1 || units[0].UnitNo != "1201" {
		t.Fatalf("units = %+v", units)
	}
}
