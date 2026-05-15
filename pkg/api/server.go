package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"hic/pkg/extraction"
	"hic/pkg/normalize"
	"hic/pkg/persistence"

	"github.com/labstack/echo/v4"
)

type Repository interface {
	ListOfferings(ctx context.Context, limit int32, qaStatus string) ([]persistence.OfferingView, error)
	ListSourceNotices(ctx context.Context, limit int32) ([]persistence.SourceNoticeView, error)
}

type Server struct {
	repo       Repository
	displayDir string
}

func New(repo Repository) *echo.Echo {
	return NewWithDisplay(repo, "display")
}

func NewWithDisplay(repo Repository, displayDir string) *echo.Echo {
	server := Server{repo: repo, displayDir: displayDir}
	e := echo.New()
	e.HideBanner = true
	e.GET("/health", server.health)
	e.GET("/offerings", server.listOfferings)
	e.GET("/notices", server.listNotices)
	e.GET("/reports/pdf-offerings", server.pdfOfferingsReport)
	if displayDir != "" {
		e.Static("/display", displayDir)
	}
	return e
}

func (s Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (s Server) listOfferings(c echo.Context) error {
	offerings, err := s.repo.ListOfferings(c.Request().Context(), parseLimit(c.QueryParam("limit"), 200), qaStatusParam(c.QueryParam("qa_status")))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, offerings)
}

func (s Server) listNotices(c echo.Context) error {
	notices, err := s.repo.ListSourceNotices(c.Request().Context(), parseLimit(c.QueryParam("limit"), 200))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, notices)
}

func (s Server) pdfOfferingsReport(c echo.Context) error {
	files := c.QueryParams()["file"]
	if len(files) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "file query parameter is required")
	}
	report, err := buildPDFOfferingsReport(files)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, report)
}

type PDFOfferingsReport struct {
	GeneratedAt string                    `json:"generated_at"`
	Totals      PDFOfferingsReportTotals  `json:"totals"`
	Files       []PDFOfferingsFileSummary `json:"files"`
	Offerings   []PDFOfferingItem         `json:"offerings"`
}

type PDFOfferingsReportTotals struct {
	Files     int `json:"files"`
	Artifacts int `json:"artifacts"`
	Offerings int `json:"offerings"`
}

type PDFOfferingsFileSummary struct {
	Path      string `json:"path"`
	Artifacts int    `json:"artifacts"`
	Offerings int    `json:"offerings"`
}

type PDFOfferingItem struct {
	File                 string   `json:"file"`
	ApplicationUnitLabel string   `json:"application_unit_label"`
	HousingName          string   `json:"housing_name"`
	ComplexName          string   `json:"complex_name"`
	UnitNo               string   `json:"unit_no"`
	ExclusiveAreaM2      *float64 `json:"exclusive_area_m2,omitempty"`
	SupplyCount          *int     `json:"supply_count,omitempty"`
	JeonseDepositKRW     *int64   `json:"jeonse_deposit_krw,omitempty"`
	DepositKRW           *int64   `json:"deposit_krw,omitempty"`
	MonthlyRentKRW       *int64   `json:"monthly_rent_krw,omitempty"`
	DormitoryFeeKRW      *int64   `json:"dormitory_fee_krw,omitempty"`
	GenderRequirement    string   `json:"gender_requirement"`
	SourceSpan           string   `json:"source_span"`
	Confidence           float64  `json:"confidence"`
}

func buildPDFOfferingsReport(files []string) (PDFOfferingsReport, error) {
	report := PDFOfferingsReport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Files:       make([]PDFOfferingsFileSummary, 0, len(files)),
	}
	for _, file := range files {
		artifacts, err := extraction.ExtractPDFArtifacts(file)
		if err != nil {
			return PDFOfferingsReport{}, err
		}
		offerings := normalizePDFOfferingsFromArtifacts(artifacts)
		report.Files = append(report.Files, PDFOfferingsFileSummary{
			Path:      file,
			Artifacts: len(artifacts),
			Offerings: len(offerings),
		})
		report.Totals.Artifacts += len(artifacts)
		for _, offering := range offerings {
			report.Offerings = append(report.Offerings, PDFOfferingItem{
				File:                 file,
				ApplicationUnitLabel: offering.ApplicationUnitLabel,
				HousingName:          offering.HousingName,
				ComplexName:          offering.ComplexName,
				UnitNo:               offering.UnitNo,
				ExclusiveAreaM2:      offering.ExclusiveAreaM2,
				SupplyCount:          offering.SupplyCount,
				JeonseDepositKRW:     offering.JeonseDepositKRW,
				DepositKRW:           offering.DepositKRW,
				MonthlyRentKRW:       offering.MonthlyRentKRW,
				DormitoryFeeKRW:      offering.DormitoryFeeKRW,
				GenderRequirement:    offering.GenderRequirement,
				SourceSpan:           offering.SourceSpan,
				Confidence:           offering.Confidence,
			})
		}
	}
	report.Totals.Files = len(files)
	report.Totals.Offerings = len(report.Offerings)
	return report, nil
}

func normalizePDFOfferingsFromArtifacts(artifacts []extraction.ExtractedArtifact) []normalize.OfferingCandidate {
	offerings := make([]normalize.OfferingCandidate, 0)
	for _, artifact := range artifacts {
		offerings = append(offerings, normalize.InferOfferingsFromPDFText(artifact)...)
	}
	offerings = append(offerings, normalize.InferOfferingsFromPDFTableRows(artifacts)...)
	return offerings
}

func parseLimit(raw string, fallback int32) int32 {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	if value > 1000 {
		return 1000
	}
	return int32(value)
}

func qaStatusParam(raw string) string {
	if raw == "" {
		return "approved"
	}
	return raw
}
