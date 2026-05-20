package extraction

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type hwpTextCommand struct {
	Name string
	Args []string
}

type hwpCommandRunner func(name string, args ...string) ([]byte, error)

func ExtractHWPTextWithSource(path string, sourceSpan string) (ExtractedArtifact, error) {
	return extractHWPTextWithCommands(path, sourceSpan, defaultHWPTextCommands(), runHWPTextCommand)
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
