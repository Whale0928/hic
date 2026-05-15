package extraction

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

func ExtractPDFText(path string) (ExtractedArtifact, error) {
	text, sourceSpan, err := readPDFPlainText(path)
	if err != nil {
		return ExtractedArtifact{}, err
	}
	return ExtractedArtifact{
		Type:          ArtifactTypePDFText,
		Extractor:     "pdf-plain-text",
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

func ExtractPDFArtifacts(path string) ([]ExtractedArtifact, error) {
	textArtifact, err := ExtractPDFText(path)
	if err != nil {
		return nil, err
	}
	artifacts := []ExtractedArtifact{textArtifact}
	artifacts = append(artifacts, ExtractPDFTableRowsFromText(textArtifact.RawText, textArtifact.SourceSpan)...)
	return artifacts, nil
}

func readPDFPlainText(path string) (string, string, error) {
	file, reader, err := pdf.Open(filepath.Clean(path))
	if err != nil {
		return "", "", fmt.Errorf("open pdf: %w", err)
	}
	defer file.Close()

	textReader, err := reader.GetPlainText()
	if err != nil {
		return "", "", fmt.Errorf("extract pdf plain text: %w", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(textReader); err != nil {
		return "", "", fmt.Errorf("read pdf plain text: %w", err)
	}

	text := cleanExtractedText(buf.String())
	return text, "pdf://" + filepath.Clean(path), nil
}

func cleanExtractedText(text string) string {
	return strings.ReplaceAll(text, "\x00", "")
}
