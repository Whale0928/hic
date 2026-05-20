package extraction

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractHWPXTextWithSource_본문텍스트Artifact를생성한다(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notice.hwpx")
	writeTestHWPX(t, path, map[string]string{
		"Contents/section0.xml": `<hp:sec xmlns:hp="http://www.hancom.co.kr/hwpml/2011/paragraph"><hp:p><hp:run><hp:t>입주자 모집공고</hp:t></hp:run></hp:p><hp:p><hp:run><hp:t>함평기산 통합공공임대</hp:t></hp:run></hp:p></hp:sec>`,
	})

	artifact, err := ExtractHWPXTextWithSource(path, "object://hic-originals/lh/20364/1-notice.hwpx")
	if err != nil {
		t.Fatalf("ExtractHWPXTextWithSource() error = %v", err)
	}

	if artifact.Type != ArtifactTypeHWPXText || artifact.Status != ArtifactStatusExtracted {
		t.Fatalf("artifact = %+v", artifact)
	}
	if artifact.SourceSpan != "object://hic-originals/lh/20364/1-notice.hwpx" {
		t.Fatalf("SourceSpan = %q", artifact.SourceSpan)
	}
	if !strings.Contains(artifact.RawText, "입주자 모집공고") || !strings.Contains(artifact.RawText, "함평기산 통합공공임대") {
		t.Fatalf("RawText = %q", artifact.RawText)
	}
}

func TestExtractHWPXTextWithSource_PreviewTextFallback을사용한다(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notice.hwpx")
	writeTestHWPX(t, path, map[string]string{
		"Preview/PrvText.txt": "잔여세대 입주자 모집공고\n신청접수",
	})

	artifact, err := ExtractHWPXTextWithSource(path, "object://hic-originals/sh/304555/1-notice.hwpx")
	if err != nil {
		t.Fatalf("ExtractHWPXTextWithSource() error = %v", err)
	}

	if artifact.RawText != "잔여세대 입주자 모집공고 신청접수" {
		t.Fatalf("RawText = %q", artifact.RawText)
	}
}

func writeTestHWPX(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	zw := zip.NewWriter(f)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip Create() error = %v", err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("zip Write() error = %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip Close() error = %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
