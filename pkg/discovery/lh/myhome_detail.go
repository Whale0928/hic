package lh

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type MyHomeNoticeDetail struct {
	DetailURL                string
	RawText                  string
	SupplyCount              *int
	SupplyCountText          string
	TargetHouseInfoText      string
	SupportLimitText         string
	RentConditionText        string
	DepositConditionText     string
	MonthlyRentConditionText string
	LeasePeriodText          string
	NoticeFileText           string
	NoticeFiles              []MyHomeNoticeFile
}

type MyHomeNoticeFile struct {
	AtchFileID string
	FileSN     string
	Filename   string
}

type MyHomeNoticeFileDocument struct {
	ContentType string
	Body        io.ReadCloser
}

func (c MyHomeClient) GetNoticeDetail(ctx context.Context, detailURL string) (MyHomeNoticeDetail, error) {
	detailURL = strings.TrimSpace(detailURL)
	if detailURL == "" {
		return MyHomeNoticeDetail{}, fmt.Errorf("myhome detail url is empty")
	}
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, detailURL, nil)
	if err != nil {
		return MyHomeNoticeDetail{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 HIC/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return MyHomeNoticeDetail{}, fmt.Errorf("request myhome detail: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return MyHomeNoticeDetail{}, fmt.Errorf("myhome detail response status %d", resp.StatusCode)
	}
	detail, err := ParseMyHomeNoticeDetailHTMLReader(resp.Body)
	if err != nil {
		return MyHomeNoticeDetail{}, err
	}
	detail.DetailURL = detailURL
	return detail, nil
}

func (c MyHomeClient) DownloadNoticeFile(ctx context.Context, file MyHomeNoticeFile) (MyHomeNoticeFileDocument, error) {
	if strings.TrimSpace(file.AtchFileID) == "" || strings.TrimSpace(file.FileSN) == "" {
		return MyHomeNoticeFileDocument{}, fmt.Errorf("myhome notice file ids are required")
	}
	form := url.Values{}
	form.Set("atchFileId", file.AtchFileID)
	form.Set("fileSn", file.FileSN)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://www.myhome.go.kr/hws/com/fms/cvplFileDownload.do", strings.NewReader(form.Encode()))
	if err != nil {
		return MyHomeNoticeFileDocument{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 HIC/1.0")
	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return MyHomeNoticeFileDocument{}, fmt.Errorf("download myhome notice file: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = resp.Body.Close()
		return MyHomeNoticeFileDocument{}, fmt.Errorf("myhome notice file response status %d", resp.StatusCode)
	}
	return MyHomeNoticeFileDocument{
		ContentType: resp.Header.Get("Content-Type"),
		Body:        resp.Body,
	}, nil
}

func ParseMyHomeNoticeDetailHTML(rawHTML string) (MyHomeNoticeDetail, error) {
	return ParseMyHomeNoticeDetailHTMLReader(strings.NewReader(rawHTML))
}

func ParseMyHomeNoticeDetailHTMLReader(reader interface {
	Read([]byte) (int, error)
}) (MyHomeNoticeDetail, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return MyHomeNoticeDetail{}, err
	}
	detail := MyHomeNoticeDetail{RawText: normalizeMyHomeDetailText(doc.Text())}
	doc.Find("tr").Each(func(_ int, row *goquery.Selection) {
		label := normalizeMyHomeDetailText(row.Find("th").First().Text())
		value := normalizeMyHomeDetailText(row.Find("td").First().Text())
		if label == "" || value == "" {
			return
		}
		switch label {
		case "공급호수":
			detail.SupplyCountText = value
			detail.SupplyCount = parseSupplyCountText(value)
		case "대상 주택 정보":
			detail.TargetHouseInfoText = value
		case "지원한도액":
			detail.SupportLimitText = value
		case "임대조건":
			detail.RentConditionText = value
			detail.DepositConditionText, detail.MonthlyRentConditionText = parseRentConditionText(value)
		case "임대기간":
			detail.LeasePeriodText = value
		case "공고문":
			detail.NoticeFileText = value
		}
	})
	doc.Find("a").Each(func(_ int, link *goquery.Selection) {
		href, ok := link.Attr("href")
		if !ok {
			return
		}
		match := myHomeNoticeFilePattern.FindStringSubmatch(href)
		if len(match) < 3 {
			return
		}
		detail.NoticeFiles = append(detail.NoticeFiles, MyHomeNoticeFile{
			AtchFileID: match[1],
			FileSN:     match[2],
			Filename:   normalizeMyHomeDetailText(link.Text()),
		})
	})
	return detail, nil
}

func (d MyHomeNoticeDetail) Content() map[string]any {
	content := map[string]any{
		"detail_url": d.DetailURL,
	}
	addString := func(key string, value string) {
		if strings.TrimSpace(value) != "" {
			content[key] = value
		}
	}
	if d.SupplyCount != nil {
		content["supply_count"] = *d.SupplyCount
	}
	addString("supply_count_text", d.SupplyCountText)
	addString("target_house_info_text", d.TargetHouseInfoText)
	addString("support_limit_text", d.SupportLimitText)
	addString("rent_condition_text", d.RentConditionText)
	addString("deposit_condition_text", d.DepositConditionText)
	addString("monthly_rent_condition_text", d.MonthlyRentConditionText)
	addString("lease_period_text", d.LeasePeriodText)
	addString("notice_file_text", d.NoticeFileText)
	if len(d.NoticeFiles) > 0 {
		content["notice_files"] = d.NoticeFiles
	}
	return content
}

func (d MyHomeNoticeDetail) JSONContent() ([]byte, error) {
	return json.Marshal(d.Content())
}

var (
	myHomeWhitespacePattern = regexp.MustCompile(`\s+`)
	myHomeNoticeFilePattern = regexp.MustCompile(`fnDownFile\('([^']+)'\s*,\s*'([^']+)'\)`)
)

func normalizeMyHomeDetailText(text string) string {
	text = strings.ReplaceAll(text, "\u00a0", " ")
	text = myHomeWhitespacePattern.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func parseRentConditionText(text string) (string, string) {
	var deposit string
	var monthly string
	for _, part := range strings.Split(text, "-") {
		part = normalizeMyHomeDetailText(strings.TrimLeft(part, "•* "))
		if part == "" || strings.Contains(part, "자세한 사항") || strings.Contains(part, "상세내역") {
			continue
		}
		switch {
		case startsWithRentConditionLabel(part, "임대보증금"):
			deposit = part
		case startsWithRentConditionLabel(part, "월임대료"):
			monthly = part
		case startsWithRentConditionLabel(part, "임대료"):
			monthly = part
		case deposit == "" && monthly == "" && looksLikeCombinedRentCondition(part):
			deposit = part
			monthly = part
		}
	}
	return deposit, monthly
}

func startsWithRentConditionLabel(text string, label string) bool {
	text = strings.TrimSpace(text)
	return strings.HasPrefix(text, label) || strings.HasPrefix(text, label+" ")
}

func looksLikeCombinedRentCondition(text string) bool {
	if strings.Contains(text, "임대보증금") && strings.Contains(text, "임대료") {
		return true
	}
	return strings.Contains(text, "시중 전세가") || strings.Contains(text, "시중 전세가격")
}
