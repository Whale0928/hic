package sh

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type ApplicationStatus string

const (
	StatusOpen    ApplicationStatus = "청약중"
	StatusPending ApplicationStatus = "접수예정"
)

type ApplicationEndpoint struct {
	Name       string
	BaseURL    string
	Path       string
	SupplyType string
}

type ApplicationEndpoints []ApplicationEndpoint

func (e ApplicationEndpoints) BySupplyType(supplyType string) (ApplicationEndpoint, bool) {
	for _, endpoint := range e {
		if endpoint.SupplyType == supplyType {
			return endpoint, true
		}
	}
	return ApplicationEndpoint{}, false
}

func (e ApplicationEndpoints) Select(supplyTypes []string) (ApplicationEndpoints, error) {
	if len(supplyTypes) == 0 {
		return e, nil
	}
	selected := make(ApplicationEndpoints, 0, len(supplyTypes))
	for _, supplyType := range supplyTypes {
		supplyType = strings.TrimSpace(supplyType)
		if supplyType == "" {
			continue
		}
		endpoint, ok := e.BySupplyType(supplyType)
		if !ok {
			return nil, fmt.Errorf("unknown SH splyTy: %s", supplyType)
		}
		selected = append(selected, endpoint)
	}
	return selected, nil
}

func DefaultApplicationEndpoints() ApplicationEndpoints {
	return ApplicationEndpoints{
		{Name: "장기전세", BaseURL: "https://www.i-sh.co.kr", Path: "/app/lay2/program/S48T560C607/m_26/appNoti/appUser_list.do", SupplyType: "01"},
		{Name: "국민임대", BaseURL: "https://www.i-sh.co.kr", Path: "/app/lay2/program/S48T1588C614/m_27/appNoti/appUser_list.do", SupplyType: "02"},
		{Name: "공공임대", BaseURL: "https://www.i-sh.co.kr", Path: "/app/lay2/program/S48T1587C612/m_14/appNoti/appUser_list.do", SupplyType: "03"},
		{Name: "매입임대", BaseURL: "https://www.i-sh.co.kr", Path: "/app/lay2/program/S48T1589C616/m_32/appNoti/appUser_list.do", SupplyType: "04"},
		{Name: "재개발임대", BaseURL: "https://www.i-sh.co.kr", Path: "/app/lay2/program/S48T1592C621/m_36/appNoti/appUser_list.do", SupplyType: "05"},
		{Name: "희망하우징", BaseURL: "https://www.i-sh.co.kr", Path: "/app/lay2/program/S48T1591C619/m_40/appNoti/appUser_list.do", SupplyType: "06"},
		{Name: "행복주택", BaseURL: "https://www.i-sh.co.kr", Path: "/app/lay2/program/S48T1594C624/m_66/appNoti/appUser_list.do", SupplyType: "07"},
		{Name: "전세임대", BaseURL: "https://www.i-sh.co.kr", Path: "/app/lay2/program/S48T572C4932/m_78/appNoti/appUser_list.do", SupplyType: "12"},
		{Name: "장기안심", BaseURL: "https://www.i-sh.co.kr", Path: "/app/lay2/program/S48T2731C2712/m_79/appNoti/appUser_list.do", SupplyType: "10"},
	}
}

func (e ApplicationEndpoint) URL() (string, error) {
	base, err := url.Parse(e.BaseURL)
	if err != nil {
		return "", fmt.Errorf("parse SH application base URL: %w", err)
	}
	u, err := base.Parse(e.Path)
	if err != nil {
		return "", fmt.Errorf("parse SH application path: %w", err)
	}
	values := u.Query()
	values.Set("splyTy", e.SupplyType)
	u.RawQuery = values.Encode()
	return u.String(), nil
}

type ApplicationNotice struct {
	RecruitNoticeCode string
	SupplyType        string
	RecruitType       string
	NoticeNoHouse     string
	RegionPriority    string
	Title             string
	SupplyCount       *int
	PostedAt          time.Time
	Status            ApplicationStatus
	RawStatus         string
}

type HTTPClient struct {
	HTTPClient *http.Client
	UserAgent  string
}

func NewHTTPClient() HTTPClient {
	return HTTPClient{
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		UserAgent:  "hic-discovery/0.1 (+SH application status discovery)",
	}
}

func (c HTTPClient) ListApplications(ctx context.Context, endpoint ApplicationEndpoint) ([]ApplicationNotice, error) {
	rawURL, err := endpoint.URL()
	if err != nil {
		return nil, err
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", firstNonEmpty(c.UserAgent, "hic-discovery/0.1"))
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch SH application list splyTy=%s: %w", endpoint.SupplyType, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch SH application list splyTy=%s: unexpected status %d", endpoint.SupplyType, resp.StatusCode)
	}
	return ParseApplicationList(resp.Body, endpoint)
}

var userSsnCheckPattern = regexp.MustCompile(`userSsnCheck\((.*)\)`)

func ParseApplicationList(r io.Reader, endpoint ApplicationEndpoint) ([]ApplicationNotice, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("parse SH application html: %w", err)
	}

	var notices []ApplicationNotice
	seen := map[string]bool{}
	doc.Find("tr").Each(func(_ int, tr *goquery.Selection) {
		cells := tr.Find("td")
		if cells.Length() < 5 {
			return
		}
		link := cells.Eq(1).Find("a[onclick*='userSsnCheck']").First()
		onclick, ok := link.Attr("onclick")
		if !ok {
			return
		}
		args := parseUserSsnCheckArgs(onclick)
		if len(args) < 7 {
			return
		}
		statusText := cleanText(cells.Eq(4).Text())
		status, ok := applicationStatus(statusText)
		if !ok {
			return
		}
		supplyCount := parseOptionalInt(cleanText(cells.Eq(2).Text()))
		postedAt, _ := time.ParseInLocation(time.DateOnly, cleanText(cells.Eq(3).Text()), time.UTC)
		notice := ApplicationNotice{
			RecruitNoticeCode: strings.TrimSpace(args[0]),
			SupplyType:        firstNonEmpty(strings.TrimSpace(args[2]), endpoint.SupplyType),
			RecruitType:       strings.TrimSpace(args[3]),
			NoticeNoHouse:     strings.TrimSpace(args[4]),
			RegionPriority:    strings.TrimSpace(args[5]),
			Title:             firstNonEmpty(strings.TrimSpace(args[6]), cleanText(link.Text())),
			SupplyCount:       supplyCount,
			PostedAt:          postedAt,
			Status:            status,
			RawStatus:         statusText,
		}
		key := notice.RecruitNoticeCode + ":" + string(notice.Status)
		if notice.RecruitNoticeCode == "" || seen[key] {
			return
		}
		seen[key] = true
		notices = append(notices, notice)
	})

	return notices, nil
}

func applicationStatus(text string) (ApplicationStatus, bool) {
	switch {
	case strings.Contains(text, string(StatusOpen)):
		return StatusOpen, true
	case strings.Contains(text, string(StatusPending)):
		return StatusPending, true
	default:
		return "", false
	}
}

func parseUserSsnCheckArgs(onclick string) []string {
	matches := userSsnCheckPattern.FindStringSubmatch(onclick)
	if len(matches) != 2 {
		return nil
	}
	raw := matches[1]
	args := make([]string, 0, 7)
	var current strings.Builder
	inQuote := false
	for _, r := range raw {
		switch r {
		case '\'':
			inQuote = !inQuote
		case ',':
			if inQuote {
				current.WriteRune(r)
			} else {
				args = append(args, strings.TrimSpace(current.String()))
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	args = append(args, strings.TrimSpace(current.String()))
	return args
}

func parseOptionalInt(text string) *int {
	cleaned := strings.ReplaceAll(strings.TrimSpace(text), ",", "")
	if cleaned == "" {
		return nil
	}
	n, err := strconv.Atoi(cleaned)
	if err != nil {
		return nil
	}
	return &n
}

func cleanText(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
