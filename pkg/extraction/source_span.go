package extraction

import "strings"

func ObjectSourceSpan(objectKey string) string {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return ""
	}
	return "object://" + objectKey
}

func UnsupportedHWPArtifact(sourceSpan string, filename string) ExtractedArtifact {
	return ExtractedArtifact{
		Type:          ArtifactTypeHWPUnsupported,
		Extractor:     "hwp-unsupported",
		Status:        ArtifactStatusUnsupported,
		SchemaVersion: "v1",
		SourceSpan:    sourceSpan,
		Content: map[string]any{
			"filename": filename,
			"reason":   "hwp text extractor is not available",
		},
		Confidence: 0,
		ErrorText:  "hwp text extractor is not available",
	}
}
