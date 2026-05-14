package extraction

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

func ExtractPDFText(path string) (ExtractedArtifact, error) {
	file, reader, err := pdf.Open(filepath.Clean(path))
	if err != nil {
		return ExtractedArtifact{}, fmt.Errorf("open pdf: %w", err)
	}
	defer file.Close()

	textReader, err := reader.GetPlainText()
	if err != nil {
		return ExtractedArtifact{}, fmt.Errorf("extract pdf plain text: %w", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(textReader); err != nil {
		return ExtractedArtifact{}, fmt.Errorf("read pdf plain text: %w", err)
	}

	text := cleanExtractedText(buf.String())
	return ExtractedArtifact{
		Type:          ArtifactTypePDFText,
		Extractor:     "pdf-plain-text",
		Status:        ArtifactStatusExtracted,
		SchemaVersion: "v1",
		SourceSpan:    "pdf://" + filepath.Clean(path),
		RawText:       text,
		Content: map[string]any{
			"chars": len([]rune(text)),
		},
		Confidence: 1,
	}, nil
}

func cleanExtractedText(text string) string {
	return strings.ReplaceAll(text, "\x00", "")
}
