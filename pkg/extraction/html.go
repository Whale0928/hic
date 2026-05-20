package extraction

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func ExtractHTMLPreview(path string) (ExtractedArtifact, error) {
	return ExtractHTMLPreviewWithSource(path, "")
}

func ExtractHTMLPreviewWithSource(path string, sourceSpan string) (ExtractedArtifact, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return ExtractedArtifact{}, fmt.Errorf("open html preview: %w", err)
	}
	defer file.Close()

	doc, err := goquery.NewDocumentFromReader(file)
	if err != nil {
		return ExtractedArtifact{}, fmt.Errorf("parse html preview: %w", err)
	}
	doc.Find("script,style,noscript").Remove()
	text := strings.TrimSpace(strings.Join(strings.Fields(doc.Text()), " "))
	if strings.TrimSpace(sourceSpan) == "" {
		sourceSpan = "html://" + filepath.Clean(path)
	}
	return ExtractedArtifact{
		Type:          ArtifactTypeHTMLPreview,
		Extractor:     "goquery-html-text",
		Status:        ArtifactStatusExtracted,
		SchemaVersion: "v1",
		SourceSpan:    sourceSpan,
		RawText:       text,
		Content: map[string]any{
			"chars": len([]rune(text)),
		},
		Confidence: 1,
	}, nil
}
