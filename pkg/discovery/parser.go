package discovery

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type BoardRow struct {
	Agency     string
	BoardKind  string
	Seq        string
	Title      string
	Department string
	PostedAt   time.Time
	ViewCount  int
	IsNew      bool
}

type BoardDetail struct {
	Title       string
	BodyText    string
	PostedAt    *time.Time
	ViewCount   int
	Attachments []AttachmentMeta
}

type AttachmentMeta struct {
	BRDID       string `json:"brdId"`
	Seq         string `json:"seq"`
	FileSeq     string `json:"fileSeq"`
	Filename    string `json:"filename"`
	OriFileName string `json:"oriFileNm"`
	Size        string `json:"size"`
	FileSize    string `json:"fileSize"`
	FileType    string `json:"fileTp"`
	PreviewPath string
}

func (a AttachmentMeta) DisplayFilename() string {
	return firstNonEmpty(a.Filename, a.OriFileName)
}

func (a AttachmentMeta) DisplaySize() string {
	return firstNonEmpty(a.Size, a.FileSize)
}

var (
	seqPattern          = regexp.MustCompile(`getDetailView\(['"]?([0-9]+)['"]?\)`)
	downListPattern     = regexp.MustCompile(`(?s)initParam\.downList\s*=\s*(\[[^\n;]*\])`)
	htmlConverterParams = regexp.MustCompile(`brd_id=([^&]+).*seq=([^&]+).*file_seq=([^&]+)`)
	spacePattern        = regexp.MustCompile(`\s+`)
)

func ParseBoardList(r io.Reader, board Board) ([]BoardRow, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("parse board list html: %w", err)
	}

	var rows []BoardRow
	doc.Find("#listTb tbody tr").Each(func(_ int, tr *goquery.Selection) {
		cells := tr.Find("td")
		if cells.Length() < 5 {
			return
		}

		link := cells.Eq(1).Find("a").First()
		href, _ := link.Attr("href")
		onclick, _ := link.Attr("onclick")
		seq := firstNonEmpty(extractSeq(onclick), extractSeq(href))
		if seq == "" {
			return
		}

		postedAt, _ := time.Parse(time.DateOnly, cleanText(cells.Eq(3).Text()))
		viewCount, _ := strconv.Atoi(stripComma(cleanText(cells.Eq(4).Text())))
		title := cleanTitle(link.Text())

		rows = append(rows, BoardRow{
			Agency:     board.Agency,
			BoardKind:  board.BoardKind,
			Seq:        seq,
			Title:      title,
			Department: cleanText(cells.Eq(2).Text()),
			PostedAt:   postedAt,
			ViewCount:  viewCount,
			IsNew:      link.Find(".icoNew").Length() > 0 || strings.Contains(link.Text(), "NEW"),
		})
	})

	return rows, nil
}

func ParseBoardDetail(r io.Reader) (BoardDetail, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return BoardDetail{}, fmt.Errorf("read detail html: %w", err)
	}

	html := normalizeHTMLBreaks(string(body))
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return BoardDetail{}, fmt.Errorf("parse board detail html: %w", err)
	}

	detail := BoardDetail{
		Title:       extractDetailTitle(doc),
		BodyText:    cleanText(doc.Find(".cont").First().Text()),
		ViewCount:   parseInt(stripComma(firstNonEmpty(extractTableValue(doc, "조회수"), extractInlineLabelValue(doc, "조회수")))),
		Attachments: parseDownList(html),
	}

	if detail.BodyText == "" {
		detail.BodyText = cleanText(doc.Find("td").Last().Text())
	}
	if posted, err := time.Parse(time.DateOnly, firstNonEmpty(extractTableValue(doc, "작성일"), extractInlineLabelValue(doc, "등록일"))); err == nil {
		detail.PostedAt = &posted
	}

	previewByKey := make(map[string]string)
	doc.Find("a[href*='htmlConverter.do']").Each(func(_ int, a *goquery.Selection) {
		href, ok := a.Attr("href")
		if !ok {
			return
		}
		matches := htmlConverterParams.FindStringSubmatch(href)
		if len(matches) != 4 {
			return
		}
		key := matches[1] + ":" + matches[2] + ":" + matches[3]
		previewByKey[key] = href
	})
	for i := range detail.Attachments {
		key := detail.Attachments[i].BRDID + ":" + detail.Attachments[i].Seq + ":" + detail.Attachments[i].FileSeq
		detail.Attachments[i].PreviewPath = previewByKey[key]
	}

	return detail, nil
}

func extractDetailTitle(doc *goquery.Document) string {
	return firstNonEmpty(
		extractTableValue(doc, "제목"),
		cleanText(doc.Find(".detailTable caption").First().Text()),
		cleanText(doc.Find(".detailTable thead th").First().Text()),
		cleanText(doc.Find("caption").First().Text()),
		cleanText(doc.Find("thead th").First().Text()),
	)
}

func extractInlineLabelValue(doc *goquery.Document, label string) string {
	var value string
	doc.Find("li").EachWithBreak(func(_ int, li *goquery.Selection) bool {
		text := cleanText(li.Text())
		for _, prefix := range []string{label + " :", label + ":", label + "："} {
			if strings.HasPrefix(text, prefix) {
				value = strings.TrimSpace(strings.TrimPrefix(text, prefix))
				return false
			}
		}
		return true
	})
	return value
}

func parseDownList(html string) []AttachmentMeta {
	matches := downListPattern.FindStringSubmatch(html)
	if len(matches) != 2 {
		return nil
	}
	var attachments []AttachmentMeta
	if err := json.Unmarshal([]byte(matches[1]), &attachments); err != nil {
		return nil
	}
	return attachments
}

func extractSeq(href string) string {
	matches := seqPattern.FindStringSubmatch(href)
	if len(matches) != 2 {
		return ""
	}
	return matches[1]
}

func extractTableValue(doc *goquery.Document, label string) string {
	var value string
	doc.Find("tr").EachWithBreak(func(_ int, tr *goquery.Selection) bool {
		found := false
		tr.Find("th").Each(func(_ int, th *goquery.Selection) {
			if cleanText(th.Text()) == label {
				found = true
			}
		})
		if !found {
			return true
		}
		value = cleanText(tr.Find("td").First().Text())
		return false
	})
	return value
}

func cleanTitle(text string) string {
	cleaned := cleanText(text)
	cleaned = strings.TrimPrefix(cleaned, "NEW")
	cleaned = strings.TrimSuffix(cleaned, "NEW")
	return strings.TrimSpace(cleaned)
}

func cleanText(text string) string {
	return strings.TrimSpace(spacePattern.ReplaceAllString(text, " "))
}

func normalizeHTMLBreaks(html string) string {
	replacer := strings.NewReplacer("<br>", " ", "<br/>", " ", "<br />", " ", "<BR>", " ", "<BR/>", " ", "<BR />", " ")
	return replacer.Replace(html)
}

func stripComma(text string) string {
	return strings.ReplaceAll(text, ",", "")
}

func parseInt(text string) int {
	n, _ := strconv.Atoi(text)
	return n
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
