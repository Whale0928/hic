package extraction

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

func ExtractHWPXText(path string) (ExtractedArtifact, error) {
	return ExtractHWPXTextWithSource(path, "")
}

func ExtractHWPXTextWithSource(path string, sourceSpan string) (ExtractedArtifact, error) {
	reader, err := zip.OpenReader(filepath.Clean(path))
	if err != nil {
		return ExtractedArtifact{}, fmt.Errorf("open hwpx: %w", err)
	}
	defer reader.Close()

	text, source := readHWPXPreviewText(reader.File)
	if text == "" {
		text, source = readHWPXSectionText(reader.File)
	}
	if text == "" {
		return ExtractedArtifact{}, fmt.Errorf("extract hwpx text: no text content")
	}
	if strings.TrimSpace(sourceSpan) == "" {
		sourceSpan = "hwpx://" + filepath.Clean(path)
	}
	return ExtractedArtifact{
		Type:          ArtifactTypeHWPXText,
		Extractor:     "hwpx-zip-xml",
		Status:        ArtifactStatusExtracted,
		SchemaVersion: "v1",
		SourceSpan:    sourceSpan,
		RawText:       compactArtifactText(text),
		Content: map[string]any{
			"chars":  len([]rune(compactArtifactText(text))),
			"source": source,
		},
		Confidence: 0.9,
	}, nil
}

func readHWPXPreviewText(files []*zip.File) (string, string) {
	for _, file := range files {
		if !strings.EqualFold(file.Name, "Preview/PrvText.txt") {
			continue
		}
		text, err := readZipText(file)
		if err != nil {
			return "", ""
		}
		return text, file.Name
	}
	return "", ""
}

func readHWPXSectionText(files []*zip.File) (string, string) {
	sections := make([]*zip.File, 0)
	for _, file := range files {
		name := filepath.ToSlash(file.Name)
		if strings.HasPrefix(name, "Contents/section") && strings.HasSuffix(strings.ToLower(name), ".xml") {
			sections = append(sections, file)
		}
	}
	sort.Slice(sections, func(i, j int) bool { return sections[i].Name < sections[j].Name })

	var texts []string
	for _, section := range sections {
		text, err := readHWPXTextElements(section)
		if err != nil {
			continue
		}
		if text != "" {
			texts = append(texts, text)
		}
	}
	if len(texts) == 0 {
		return "", ""
	}
	return strings.Join(texts, " "), "Contents/section*.xml"
}

func readHWPXTextElements(file *zip.File) (string, error) {
	rc, err := file.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	decoder := xml.NewDecoder(rc)
	var texts []string
	inText := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		switch value := token.(type) {
		case xml.StartElement:
			if value.Name.Local == "t" {
				inText = true
			}
		case xml.EndElement:
			if value.Name.Local == "t" {
				inText = false
			}
		case xml.CharData:
			if inText {
				texts = append(texts, string([]byte(value)))
			}
		}
	}
	return strings.Join(texts, " "), nil
}

func readZipText(file *zip.File) (string, error) {
	rc, err := file.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(rc); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func compactArtifactText(text string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(strings.ReplaceAll(text, "\x00", "")), " "))
}
