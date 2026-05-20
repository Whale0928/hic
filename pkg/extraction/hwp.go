package extraction

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"github.com/richardlehane/mscfb"
)

type hwpTextCommand struct {
	Name string
	Args []string
}

type hwpCommandRunner func(name string, args ...string) ([]byte, error)

func ExtractHWPTextWithSource(path string, sourceSpan string) (ExtractedArtifact, error) {
	return extractHWPTextWithCommands(path, sourceSpan, defaultHWPTextCommands(), runHWPTextCommand)
}

func ExtractHWPArtifactsWithSource(path string, sourceSpan string) ([]ExtractedArtifact, error) {
	textArtifact, err := ExtractHWPTextWithSource(path, sourceSpan)
	if err != nil {
		return nil, err
	}
	artifacts := []ExtractedArtifact{textArtifact}
	if textArtifact.Type == ArtifactTypeHWPText {
		artifacts = append(artifacts, ExtractPDFTableRowsFromText(textArtifact.RawText, textArtifact.SourceSpan)...)
	}
	return artifacts, nil
}

func defaultHWPTextCommands() []hwpTextCommand {
	return []hwpTextCommand{
		{Name: "hwp5txt", Args: []string{"{file}"}},
	}
}

func extractHWPTextWithCommands(path string, sourceSpan string, commands []hwpTextCommand, runner hwpCommandRunner) (ExtractedArtifact, error) {
	if strings.TrimSpace(sourceSpan) == "" {
		sourceSpan = "hwp://" + filepath.Clean(path)
	}
	if len(commands) == 0 {
		return unsupportedHWPWithReason(sourceSpan, filepath.Base(path), "no hwp text extractor command configured"), nil
	}

	var failures []string
	for _, command := range commands {
		args := make([]string, 0, len(command.Args))
		for _, arg := range command.Args {
			args = append(args, strings.ReplaceAll(arg, "{file}", filepath.Clean(path)))
		}
		output, err := runner(command.Name, args...)
		text := compactArtifactText(string(output))
		if err == nil && text != "" {
			return ExtractedArtifact{
				Type:          ArtifactTypeHWPText,
				Extractor:     command.Name,
				Status:        ArtifactStatusExtracted,
				SchemaVersion: "v1",
				SourceSpan:    sourceSpan,
				RawText:       text,
				Content: map[string]any{
					"chars":   len([]rune(text)),
					"command": command.Name,
				},
				Confidence: 0.8,
			}, nil
		}
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %s", command.Name, strings.TrimSpace(err.Error())))
			continue
		}
		failures = append(failures, fmt.Sprintf("%s: empty output", command.Name))
	}
	if artifact, err := extractHWPBodyTextWithSource(path, sourceSpan); err == nil && strings.TrimSpace(artifact.RawText) != "" {
		return artifact, nil
	} else if err != nil {
		failures = append(failures, "hwp5-ole-bodytext: "+err.Error())
	}
	return unsupportedHWPWithReason(sourceSpan, filepath.Base(path), strings.Join(failures, "; ")), nil
}

func runHWPTextCommand(name string, args ...string) ([]byte, error) {
	output, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text != "" {
			return output, fmt.Errorf("%w: %s", err, text)
		}
		return output, err
	}
	return output, nil
}

func unsupportedHWPWithReason(sourceSpan string, filename string, reason string) ExtractedArtifact {
	artifact := UnsupportedHWPArtifact(sourceSpan, filename)
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "hwp text extractor is not available"
	}
	artifact.Content["reason"] = reason
	artifact.ErrorText = reason
	return artifact
}

func extractHWPBodyTextWithSource(path string, sourceSpan string) (ExtractedArtifact, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return ExtractedArtifact{}, fmt.Errorf("open hwp compound file: %w", err)
	}
	defer file.Close()

	reader, err := mscfb.New(file)
	if err != nil {
		return ExtractedArtifact{}, fmt.Errorf("read hwp compound file: %w", err)
	}

	var sections []string
	for _, entry := range reader.File {
		if !isHWPBodyTextSection(entry) {
			continue
		}
		data, err := io.ReadAll(entry)
		if err != nil {
			return ExtractedArtifact{}, fmt.Errorf("read hwp stream %s: %w", entry.Name, err)
		}
		text := bestHWPBodyText(data)
		if text != "" {
			sections = append(sections, text)
		}
	}
	text := compactArtifactText(strings.Join(sections, "\n"))
	if text == "" {
		return ExtractedArtifact{}, fmt.Errorf("no BodyText/Section text decoded")
	}
	return ExtractedArtifact{
		Type:          ArtifactTypeHWPText,
		Extractor:     "hwp5-ole-bodytext",
		Status:        ArtifactStatusExtracted,
		SchemaVersion: "v1",
		SourceSpan:    sourceSpan,
		RawText:       text,
		Content: map[string]any{
			"chars": len([]rune(text)),
		},
		Confidence: 0.68,
	}, nil
}

func isHWPBodyTextSection(entry *mscfb.File) bool {
	if entry == nil || entry.Size <= 0 || !strings.HasPrefix(entry.Name, "Section") {
		return false
	}
	for _, pathPart := range entry.Path {
		if pathPart == "BodyText" {
			return true
		}
	}
	return false
}

func bestHWPBodyText(data []byte) string {
	best := decodeHWPBodyTextRecords(data)
	for _, inflated := range inflateHWPStreamCandidates(data) {
		if text := decodeHWPBodyTextRecords(inflated); len([]rune(text)) > len([]rune(best)) {
			best = text
		}
	}
	return best
}

func inflateHWPStreamCandidates(data []byte) [][]byte {
	var out [][]byte
	if inflated, ok := readCompressedHWPStream(data, func(reader io.Reader) (io.ReadCloser, error) {
		return zlib.NewReader(reader)
	}); ok {
		out = append(out, inflated)
	}
	if inflated, ok := readCompressedHWPStream(data, func(reader io.Reader) (io.ReadCloser, error) {
		return flate.NewReader(reader), nil
	}); ok {
		out = append(out, inflated)
	}
	return out
}

func readCompressedHWPStream(data []byte, open func(io.Reader) (io.ReadCloser, error)) ([]byte, bool) {
	reader, err := open(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	defer reader.Close()
	inflated, err := io.ReadAll(reader)
	if err != nil || len(inflated) == 0 {
		return nil, false
	}
	return inflated, true
}

func decodeHWPBodyTextRecords(data []byte) string {
	var b strings.Builder
	for offset := 0; offset+4 <= len(data); {
		header := binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4
		tagID := header & 0x3ff
		size := int((header >> 20) & 0xfff)
		if size == 0xfff {
			if offset+4 > len(data) {
				break
			}
			size = int(binary.LittleEndian.Uint32(data[offset : offset+4]))
			offset += 4
		}
		if size < 0 || offset+size > len(data) {
			break
		}
		payload := data[offset : offset+size]
		offset += size
		if tagID != 67 {
			continue
		}
		if text := decodeUTF16LEText(payload); text != "" {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(text)
		}
	}
	return compactArtifactText(b.String())
}

func decodeUTF16LEText(data []byte) string {
	if len(data) < 2 {
		return ""
	}
	words := make([]uint16, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 {
		words = append(words, binary.LittleEndian.Uint16(data[i:i+2]))
	}
	var b strings.Builder
	for _, r := range utf16.Decode(words) {
		switch {
		case r == 0:
			continue
		case r == '\r' || r == '\n' || r == '\t':
			b.WriteByte(' ')
		case r < 32:
			continue
		default:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
