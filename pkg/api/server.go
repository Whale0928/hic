package api

import (
	"context"
	"net/http"
	"strconv"

	"hic/pkg/persistence"

	"github.com/labstack/echo/v4"
)

type Repository interface {
	ListHousingUnits(ctx context.Context, limit int32) ([]persistence.HousingUnitView, error)
	ListSourceNotices(ctx context.Context, limit int32) ([]persistence.SourceNoticeView, error)
}

type Server struct {
	repo Repository
}

func New(repo Repository) *echo.Echo {
	server := Server{repo: repo}
	e := echo.New()
	e.HideBanner = true
	e.GET("/health", server.health)
	e.GET("/units", server.listUnits)
	e.GET("/notices", server.listNotices)
	return e
}

func (s Server) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (s Server) listUnits(c echo.Context) error {
	units, err := s.repo.ListHousingUnits(c.Request().Context(), parseLimit(c.QueryParam("limit"), 200))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, units)
}

func (s Server) listNotices(c echo.Context) error {
	notices, err := s.repo.ListSourceNotices(c.Request().Context(), parseLimit(c.QueryParam("limit"), 200))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, notices)
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
