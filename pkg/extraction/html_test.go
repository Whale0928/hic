package extraction

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractHTMLPreviewWithSource_본문텍스트Artifact를생성한다(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preview.html")
	if err := os.WriteFile(path, []byte(`<!doctype html>
<html><body>
	<h1>입주자 모집공고</h1>
	<table><tr><th>주택명</th><td>함평기산</td></tr></table>
	<script>ignored()</script>
</body></html>`), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	artifact, err := ExtractHTMLPreviewWithSource(path, "object://hic-artifacts/sh/304295/1-preview.html")
	if err != nil {
		t.Fatalf("ExtractHTMLPreviewWithSource() error = %v", err)
	}

	if artifact.Type != ArtifactTypeHTMLPreview || artifact.Status != ArtifactStatusExtracted {
		t.Fatalf("artifact type/status = %+v", artifact)
	}
	if artifact.SourceSpan != "object://hic-artifacts/sh/304295/1-preview.html" {
		t.Fatalf("SourceSpan = %q", artifact.SourceSpan)
	}
	if !strings.Contains(artifact.RawText, "입주자 모집공고") || !strings.Contains(artifact.RawText, "함평기산") {
		t.Fatalf("RawText = %q", artifact.RawText)
	}
	if strings.Contains(artifact.RawText, "ignored") {
		t.Fatalf("RawText includes script content: %q", artifact.RawText)
	}
}
