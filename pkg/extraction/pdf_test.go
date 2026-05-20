package extraction

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestExtractPDFText_텍스트Artifact를생성한다(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.pdf")
	if err := os.WriteFile(path, minimalPDF("Hello HIC"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	artifact, err := ExtractPDFText(path)
	if err != nil {
		t.Fatalf("ExtractPDFText() error = %v", err)
	}
	if artifact.Type != ArtifactTypePDFText || artifact.Status != ArtifactStatusExtracted {
		t.Fatalf("artifact type/status = %+v", artifact)
	}
	if !strings.Contains(artifact.RawText, "Hello HIC") {
		t.Fatalf("RawText = %q, want substring Hello HIC", artifact.RawText)
	}
	if artifact.SourceSpan == "" || artifact.SchemaVersion == "" {
		t.Fatalf("artifact missing evidence metadata: %+v", artifact)
	}
}

func TestExtractPDFArtifactsWithSource_ObjectKey를SourceSpan으로사용한다(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.pdf")
	if err := os.WriteFile(path, minimalPDF("Hello HIC"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	artifacts, err := ExtractPDFArtifactsWithSource(path, "object://hic-originals/sh/304295/1-notice.pdf")
	if err != nil {
		t.Fatalf("ExtractPDFArtifactsWithSource() error = %v", err)
	}

	if len(artifacts) == 0 {
		t.Fatalf("artifacts is empty")
	}
	if artifacts[0].SourceSpan != "object://hic-originals/sh/304295/1-notice.pdf" {
		t.Fatalf("SourceSpan = %q", artifacts[0].SourceSpan)
	}
}

func TestCleanExtractedText_nullByte를제거한다(t *testing.T) {
	got := cleanExtractedText("a\x00b")
	if got != "ab" {
		t.Fatalf("cleanExtractedText() = %q, want ab", got)
	}
}

func minimalPDF(text string) []byte {
	objects := []string{
		`<< /Type /Catalog /Pages 2 0 R >>`,
		`<< /Type /Pages /Kids [3 0 R] /Count 1 >>`,
		`<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>`,
		`<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>`,
		"<< /Length " + strconv.Itoa(len("BT /F1 24 Tf 100 700 Td ("+text+") Tj ET")) + " >>\nstream\nBT /F1 24 Tf 100 700 Td (" + text + ") Tj ET\nendstream",
	}

	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = b.Len()
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(" 0 obj\n")
		b.WriteString(obj)
		b.WriteString("\nendobj\n")
	}
	xrefOffset := b.Len()
	b.WriteString("xref\n")
	b.WriteString("0 ")
	b.WriteString(strconv.Itoa(len(objects) + 1))
	b.WriteString("\n")
	b.WriteString("0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		b.WriteString(leftPadInt(offsets[i], 10))
		b.WriteString(" 00000 n \n")
	}
	b.WriteString("trailer\n")
	b.WriteString("<< /Size ")
	b.WriteString(strconv.Itoa(len(objects) + 1))
	b.WriteString(" /Root 1 0 R >>\n")
	b.WriteString("startxref\n")
	b.WriteString(strconv.Itoa(xrefOffset))
	b.WriteString("\n%%EOF\n")
	return []byte(b.String())
}

func leftPadInt(n int, width int) string {
	value := strconv.Itoa(n)
	for len(value) < width {
		value = "0" + value
	}
	return value
}
