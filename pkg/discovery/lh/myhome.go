package lh

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type MyHomeEndpoint string

const (
	MyHomeRental MyHomeEndpoint = "rsdtRcritNtcList"
	MyHomeSale   MyHomeEndpoint = "ltRsdtRcritNtcList"
)

type MyHomeClient struct {
	BaseURL      string
	ServiceKey   string
	HTTPClient   *http.Client
	MaxAttempts  int
	RetryBackoff time.Duration
}

type MyHomePage struct {
	TotalCount int
	Items      []MyHomeNoticeItem
}

type MyHomeNoticeItem struct {
	NoticeID                 string
	HouseSN                  int
	Status                   string
	Title                    string
	Agency                   string
	HouseType                string
	SupplyType               string
	PostedDate               string
	WinnerDate               string
	SupplyCount              *int
	SupplyCountText          string
	Reference                string
	SourceURL                string
	DetailURL                string
	MobileURL                string
	ComplexName              string
	Province                 string
	City                     string
	Address                  string
	LegalDong                string
	PNU                      string
	HeatingMethod            string
	TotalHousehold           string
	DepositKRW               *int64
	MonthlyRent              *int64
	ContractPaymentKRW       *int64
	InterimPaymentKRW        *int64
	BalancePaymentKRW        *int64
	ApplicationBeg           string
	ApplicationEnd           string
	TargetHouseInfoText      string
	SupportLimitText         string
	RentConditionText        string
	DepositConditionText     string
	MonthlyRentConditionText string
	LeasePeriodText          string
	Raw                      map[string]any
}

func (i MyHomeNoticeItem) SourceSeq() string {
	if i.HouseSN <= 0 {
		return strings.TrimSpace(i.NoticeID)
	}
	return fmt.Sprintf("%s:%d", strings.TrimSpace(i.NoticeID), i.HouseSN)
}

func MyHomeSourceSpan(endpoint MyHomeEndpoint, item MyHomeNoticeItem) string {
	seq := strings.ReplaceAll(item.SourceSeq(), ":", "/")
	return fmt.Sprintf("myhome://%s/%s", endpoint, seq)
}

func (c MyHomeClient) ListNotices(ctx context.Context, endpoint MyHomeEndpoint, pageNo int, numRows int) (MyHomePage, error) {
	if pageNo <= 0 {
		pageNo = 1
	}
	if numRows <= 0 {
		numRows = 200
	}
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://apis.data.go.kr/1613000/HWSPR02"
	}
	requestURL, err := url.Parse(baseURL + "/" + string(endpoint))
	if err != nil {
		return MyHomePage{}, err
	}
	query := requestURL.Query()
	query.Set("serviceKey", c.ServiceKey)
	query.Set("pageNo", strconv.Itoa(pageNo))
	query.Set("numOfRows", strconv.Itoa(numRows))
	requestURL.RawQuery = query.Encode()

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	maxAttempts := c.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	var resp *http.Response
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
		if err != nil {
			return MyHomePage{}, err
		}
		resp, err = client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request myhome %s page=%d: %s", endpoint, pageNo, redactSecret(err.Error(), c.ServiceKey))
			if attempt < maxAttempts {
				sleepMyHomeRetry(ctx, c.RetryBackoff)
				continue
			}
			return MyHomePage{}, lastErr
		}
		if !isRetryableMyHomeStatus(resp.StatusCode) || attempt == maxAttempts {
			break
		}
		_ = resp.Body.Close()
		lastErr = fmt.Errorf("myhome response status %d", resp.StatusCode)
		sleepMyHomeRetry(ctx, c.RetryBackoff)
	}
	if resp == nil {
		if lastErr != nil {
			return MyHomePage{}, lastErr
		}
		return MyHomePage{}, fmt.Errorf("request myhome %s page=%d failed", endpoint, pageNo)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return MyHomePage{}, fmt.Errorf("myhome response status %d", resp.StatusCode)
	}

	var root struct {
		Response struct {
			Header struct {
				ResultCode string `json:"resultCode"`
				ResultMsg  string `json:"resultMsg"`
			} `json:"header"`
			Body struct {
				TotalCount any             `json:"totalCount"`
				Item       json.RawMessage `json:"item"`
			} `json:"body"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&root); err != nil {
		return MyHomePage{}, fmt.Errorf("decode myhome response: %w", err)
	}
	if root.Response.Header.ResultCode != "" && root.Response.Header.ResultCode != "00" {
		return MyHomePage{}, fmt.Errorf("myhome result %s: %s", root.Response.Header.ResultCode, root.Response.Header.ResultMsg)
	}

	items, err := parseMyHomeItems(root.Response.Body.Item)
	if err != nil {
		return MyHomePage{}, err
	}
	return MyHomePage{
		TotalCount: intFromAny(root.Response.Body.TotalCount),
		Items:      items,
	}, nil
}

func isRetryableMyHomeStatus(statusCode int) bool {
	return statusCode == http.StatusRequestTimeout || statusCode == http.StatusTooManyRequests || statusCode >= 500
}

func sleepMyHomeRetry(ctx context.Context, backoff time.Duration) {
	if backoff <= 0 {
		return
	}
	timer := time.NewTimer(backoff)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func redactSecret(text string, secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return text
	}
	redacted := strings.ReplaceAll(text, secret, "[REDACTED]")
	escaped := url.QueryEscape(secret)
	if escaped != secret {
		redacted = strings.ReplaceAll(redacted, escaped, "[REDACTED]")
	}
	return redacted
}

func parseMyHomeItems(raw json.RawMessage) ([]MyHomeNoticeItem, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var rows []map[string]any
	if raw[0] == '[' {
		if err := json.Unmarshal(raw, &rows); err != nil {
			return nil, fmt.Errorf("decode myhome item array: %w", err)
		}
	} else {
		var row map[string]any
		if err := json.Unmarshal(raw, &row); err != nil {
			return nil, fmt.Errorf("decode myhome item: %w", err)
		}
		rows = []map[string]any{row}
	}

	items := make([]MyHomeNoticeItem, 0, len(rows))
	for _, row := range rows {
		supplyCountText := stringFromAny(row["suplyHoCo"])
		items = append(items, MyHomeNoticeItem{
			NoticeID:           stringFromAny(row["pblancId"]),
			HouseSN:            intFromAny(row["houseSn"]),
			Status:             stringFromAny(row["sttusNm"]),
			Title:              stringFromAny(row["pblancNm"]),
			Agency:             stringFromAny(row["suplyInsttNm"]),
			HouseType:          stringFromAny(row["houseTyNm"]),
			SupplyType:         stringFromAny(row["suplyTyNm"]),
			PostedDate:         stringFromAny(row["rcritPblancDe"]),
			WinnerDate:         stringFromAny(row["przwnerPresnatnDe"]),
			SupplyCount:        firstPositiveIntPtr(intPtrFromAny(row["sumSuplyCo"]), parseSupplyCountText(supplyCountText)),
			SupplyCountText:    supplyCountText,
			Reference:          stringFromAny(row["refrnc"]),
			SourceURL:          stringFromAny(row["url"]),
			DetailURL:          stringFromAny(row["pcUrl"]),
			MobileURL:          stringFromAny(row["mobileUrl"]),
			ComplexName:        stringFromAny(row["hsmpNm"]),
			Province:           stringFromAny(row["brtcNm"]),
			City:               stringFromAny(row["signguNm"]),
			Address:            stringFromAny(row["fullAdres"]),
			LegalDong:          stringFromAny(row["refrnLegaldongNm"]),
			PNU:                stringFromAny(row["pnu"]),
			HeatingMethod:      stringFromAny(row["heatMthdNm"]),
			TotalHousehold:     stringFromAny(row["totHshldCo"]),
			DepositKRW:         positiveInt64PtrFromAny(row["rentGtn"]),
			MonthlyRent:        positiveInt64PtrFromAny(row["mtRntchrg"]),
			ContractPaymentKRW: positiveInt64PtrFromAny(row["enty"]),
			InterimPaymentKRW:  positiveInt64PtrFromAny(row["prtpay"]),
			BalancePaymentKRW:  positiveInt64PtrFromAny(row["surlus"]),
			ApplicationBeg:     stringFromAny(row["beginDe"]),
			ApplicationEnd:     stringFromAny(row["endDe"]),
			Raw:                row,
		})
	}
	return items, nil
}

func (i MyHomeNoticeItem) WithDetail(detail MyHomeNoticeDetail) MyHomeNoticeItem {
	if (i.SupplyCount == nil || *i.SupplyCount <= 0) && detail.SupplyCount != nil {
		i.SupplyCount = cloneInt(detail.SupplyCount)
	}
	if strings.TrimSpace(i.SupplyCountText) == "" {
		i.SupplyCountText = detail.SupplyCountText
	}
	if strings.TrimSpace(i.TargetHouseInfoText) == "" {
		i.TargetHouseInfoText = detail.TargetHouseInfoText
	}
	if strings.TrimSpace(i.SupportLimitText) == "" {
		i.SupportLimitText = detail.SupportLimitText
	}
	if strings.TrimSpace(i.RentConditionText) == "" {
		i.RentConditionText = detail.RentConditionText
	}
	if strings.TrimSpace(i.DepositConditionText) == "" {
		i.DepositConditionText = detail.DepositConditionText
	}
	if strings.TrimSpace(i.MonthlyRentConditionText) == "" {
		i.MonthlyRentConditionText = detail.MonthlyRentConditionText
	}
	if strings.TrimSpace(i.LeasePeriodText) == "" {
		i.LeasePeriodText = detail.LeasePeriodText
	}
	return i
}

func stringFromAny(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return ""
	}
}

func intFromAny(value any) int {
	if ptr := intPtrFromAny(value); ptr != nil {
		return *ptr
	}
	return 0
}

func intPtrFromAny(value any) *int {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case float64:
		out := int(v)
		return &out
	case string:
		v = strings.ReplaceAll(strings.TrimSpace(v), ",", "")
		if v == "" {
			return nil
		}
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil
		}
		return &parsed
	default:
		return nil
	}
}

func int64PtrFromAny(value any) *int64 {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case float64:
		out := int64(v)
		return &out
	case string:
		v = strings.ReplaceAll(strings.TrimSpace(v), ",", "")
		if v == "" {
			return nil
		}
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil
		}
		return &parsed
	default:
		return nil
	}
}

func positiveInt64PtrFromAny(value any) *int64 {
	ptr := int64PtrFromAny(value)
	if ptr == nil || *ptr <= 0 {
		return nil
	}
	return ptr
}

var supplyCountPattern = regexp.MustCompile(`([0-9][0-9,]*)\s*호`)

func parseSupplyCountText(text string) *int {
	match := supplyCountPattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return nil
	}
	parsed, err := strconv.Atoi(strings.ReplaceAll(match[1], ",", ""))
	if err != nil || parsed <= 0 {
		return nil
	}
	return &parsed
}

func firstPositiveIntPtr(values ...*int) *int {
	for _, value := range values {
		if value != nil && *value > 0 {
			return value
		}
	}
	return nil
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	out := *value
	return &out
}
