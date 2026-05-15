package api

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"hic/pkg/persistence"

	"github.com/labstack/echo/v4"
)

type Repository interface {
	ListOfferings(ctx context.Context, limit int32, qaStatus string) ([]persistence.OfferingView, error)
	ListSourceNotices(ctx context.Context, limit int32) ([]persistence.SourceNoticeView, error)
}

type Server struct {
	repo         Repository
	resourcesDir string
}

func New(repo Repository) *echo.Echo {
	return NewWithResources(repo, "resources")
}

func NewWithResources(repo Repository, resourcesDir string) *echo.Echo {
	server := Server{repo: repo, resourcesDir: resourcesDir}
	e := echo.New()
	e.HideBanner = true
	e.GET("/health", server.health)
	e.GET("/offerings", server.listOfferings)
	e.GET("/notices", server.listNotices)
	if resourcesDir != "" {
		e.Static("/resources", resourcesDir)
		e.GET("/reports/pdf-offerings", server.pdfOfferingsReport)
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
	path := filepath.Join(s.resourcesDir, "pdf-offerings.json")
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return echo.NewHTTPError(http.StatusNotFound, "pdf offerings report not found")
		}
		return err
	}
	return c.JSONBlob(http.StatusOK, body)
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
