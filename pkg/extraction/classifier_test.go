package extraction

import "testing"

func TestClassifyAttachment_첨부역할을분류한다(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     AttachmentKind
	}{
		{name: "공고문PDF", filename: "[공고문] 휘경마을 두레주택 모집공고.pdf", want: AttachmentKindNoticePDF},
		{name: "주택목록XLSX", filename: "붙임1 주택목록.xlsx", want: AttachmentKindOfferingListXLSX},
		{name: "당첨자명단", filename: "청년 매입임대주택 당첨자 명단.xlsx", want: AttachmentKindApplicantOrWinnerFile},
		{name: "예비자명단", filename: "예비자 명단.pdf", want: AttachmentKindApplicantOrWinnerFile},
		{name: "신청서", filename: "입주신청서.hwp", want: AttachmentKindApplicationForm},
		{name: "HWP공고", filename: "모집공고.hwp", want: AttachmentKindUnsupported},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClassifyAttachment(tt.filename); got != tt.want {
				t.Fatalf("ClassifyAttachment(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestExtractedArtifact_필수메타데이터를표현한다(t *testing.T) {
	artifact := ExtractedArtifact{
		Type:          ArtifactTypeXLSXRow,
		Extractor:     "excelize",
		Status:        ArtifactStatusExtracted,
		SourceSheet:   "주택목록",
		SourceRow:     7,
		SourceSpan:    "xlsx://주택목록!7",
		SchemaVersion: "v1",
		Confidence:    1,
	}

	if artifact.SourceSpan == "" || artifact.SchemaVersion == "" {
		t.Fatalf("artifact missing evidence metadata: %+v", artifact)
	}
}
