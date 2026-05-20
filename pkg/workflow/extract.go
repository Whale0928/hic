package workflow

import (
	"path/filepath"
	"strings"

	"hic/pkg/extraction"
)

type PreservedAttachmentRef struct {
	ObjectKey string
	Filename  string
	Kind      extraction.AttachmentKind
}

func ExtractPreservedAttachment(objectStore extraction.LocalObjectStore, attachment PreservedAttachmentRef) ([]extraction.ExtractedArtifact, error) {
	path, err := objectStore.PathForKey(attachment.ObjectKey)
	if err != nil {
		return nil, err
	}
	sourceSpan := extraction.ObjectSourceSpan(attachment.ObjectKey)
	switch attachment.Kind {
	case extraction.AttachmentKindNoticePDF, extraction.AttachmentKindSchedulePDF:
		return extraction.ExtractPDFArtifactsWithSource(path, sourceSpan)
	case extraction.AttachmentKindOfferingListXLSX:
		return extraction.ExtractXLSXRowsWithSource(path, sourceSpan)
	case extraction.AttachmentKindNoticeHWP:
		if strings.EqualFold(filepath.Ext(firstNonEmpty(attachment.Filename, attachment.ObjectKey)), ".hwpx") {
			return singleArtifact(extraction.ExtractHWPXTextWithSource(path, sourceSpan))
		}
		return singleArtifact(extraction.ExtractHWPTextWithSource(path, sourceSpan))
	default:
		return nil, nil
	}
}

func ExtractPreservedPreview(objectStore extraction.LocalObjectStore, objectKey string) ([]extraction.ExtractedArtifact, error) {
	if strings.TrimSpace(objectKey) == "" {
		return nil, nil
	}
	path, err := objectStore.PathForKey(objectKey)
	if err != nil {
		return nil, err
	}
	return singleArtifact(extraction.ExtractHTMLPreviewWithSource(path, extraction.ObjectSourceSpan(objectKey)))
}

func singleArtifact(artifact extraction.ExtractedArtifact, err error) ([]extraction.ExtractedArtifact, error) {
	if err != nil {
		return nil, err
	}
	return []extraction.ExtractedArtifact{artifact}, nil
}
