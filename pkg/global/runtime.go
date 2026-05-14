package global

import "time"

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}

type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

type Site struct {
	Agency    string
	BoardKind string
	Name      string
	BaseURL   string
}

type SiteRegistry interface {
	Get(agency string, boardKind string) (Site, bool)
	List() []Site
}
