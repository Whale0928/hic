package extraction

import (
	"encoding/binary"
	"errors"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestExtractHWPTextWithCommands_외부도구출력을텍스트Artifact로만든다(t *testing.T) {
	runner := func(name string, args ...string) ([]byte, error) {
		if name != "hwp5txt" {
			t.Fatalf("name = %q, want hwp5txt", name)
		}
		if len(args) != 1 || args[0] != "/tmp/notice.hwp" {
			t.Fatalf("args = %+v", args)
		}
		return []byte("입주자 모집공고\n신청접수"), nil
	}

	artifact, err := extractHWPTextWithCommands("/tmp/notice.hwp", "object://hic-originals/sh/304295/1-notice.hwp", []hwpTextCommand{
		{Name: "hwp5txt", Args: []string{"{file}"}},
	}, runner)
	if err != nil {
		t.Fatalf("extractHWPTextWithCommands() error = %v", err)
	}

	if artifact.Type != ArtifactTypeHWPText || artifact.Status != ArtifactStatusExtracted {
		t.Fatalf("artifact = %+v", artifact)
	}
	if artifact.Extractor != "hwp5txt" {
		t.Fatalf("Extractor = %q", artifact.Extractor)
	}
	if artifact.SourceSpan != "object://hic-originals/sh/304295/1-notice.hwp" {
		t.Fatalf("SourceSpan = %q", artifact.SourceSpan)
	}
	if artifact.RawText != "입주자 모집공고 신청접수" {
		t.Fatalf("RawText = %q", artifact.RawText)
	}
}

func TestExtractHWPTextWithCommands_모든도구실패시UnsupportedArtifact를반환한다(t *testing.T) {
	runner := func(name string, args ...string) ([]byte, error) {
		return nil, errors.New("not installed")
	}

	artifact, err := extractHWPTextWithCommands("/tmp/notice.hwp", "object://hic-originals/sh/304295/1-notice.hwp", []hwpTextCommand{
		{Name: "hwp5txt", Args: []string{"{file}"}},
	}, runner)
	if err != nil {
		t.Fatalf("extractHWPTextWithCommands() error = %v", err)
	}

	if artifact.Type != ArtifactTypeHWPUnsupported || artifact.Status != ArtifactStatusUnsupported {
		t.Fatalf("artifact = %+v", artifact)
	}
	if !strings.Contains(artifact.ErrorText, "not installed") {
		t.Fatalf("ErrorText = %q", artifact.ErrorText)
	}
}

func TestDecodeHWPBodyTextRecords_ParaText레코드를UTF16텍스트로추출한다(t *testing.T) {
	payload := utf16LEBytes("마곡 도전숙\n공급대상 7호")
	record := hwpRecord(67, payload)

	text := decodeHWPBodyTextRecords(record)

	if text != "마곡 도전숙 공급대상 7호" {
		t.Fatalf("decodeHWPBodyTextRecords() = %q", text)
	}
}

func hwpRecord(tag uint32, payload []byte) []byte {
	header := tag | uint32(len(payload))<<20
	record := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint32(record[:4], header)
	copy(record[4:], payload)
	return record
}

func utf16LEBytes(text string) []byte {
	encoded := utf16.Encode([]rune(text))
	out := make([]byte, len(encoded)*2)
	for i, value := range encoded {
		binary.LittleEndian.PutUint16(out[i*2:], value)
	}
	return out
}
