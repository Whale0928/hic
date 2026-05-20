package extraction

type AttachmentKind string

const (
	AttachmentKindNoticePDF             AttachmentKind = "notice_pdf"
	AttachmentKindNoticeHWP             AttachmentKind = "notice_hwp"
	AttachmentKindOfferingListXLSX      AttachmentKind = "offering_list_xlsx"
	AttachmentKindSchedulePDF           AttachmentKind = "schedule_pdf"
	AttachmentKindApplicantOrWinnerFile AttachmentKind = "applicant_or_winner_file"
	AttachmentKindApplicationForm       AttachmentKind = "application_form"
	AttachmentKindUnsupported           AttachmentKind = "unsupported"
)

type ArtifactType string

const (
	ArtifactTypePDFText        ArtifactType = "pdf_text"
	ArtifactTypePDFTableRow    ArtifactType = "pdf_table_row"
	ArtifactTypeXLSXRow        ArtifactType = "xlsx_row"
	ArtifactTypeHWPText        ArtifactType = "hwp_text"
	ArtifactTypeHWPUnsupported ArtifactType = "hwp_unsupported"
	ArtifactTypeMyHomeAPIItem  ArtifactType = "myhome_api_item"
	ArtifactTypeMyHomeDetail   ArtifactType = "myhome_detail_html"
	ArtifactTypeHTMLPreview    ArtifactType = "html_preview"
	ArtifactTypeHWPXText       ArtifactType = "hwpx_text"
)

type ArtifactStatus string

const (
	ArtifactStatusExtracted   ArtifactStatus = "extracted"
	ArtifactStatusUnsupported ArtifactStatus = "unsupported"
	ArtifactStatusFailed      ArtifactStatus = "failed"
)

type ExtractedArtifact struct {
	Type          ArtifactType
	Extractor     string
	Status        ArtifactStatus
	SchemaVersion string
	SourceSheet   string
	SourceRow     int
	SourceCell    string
	SourcePage    int
	SourceSpan    string
	RawText       string
	Content       map[string]any
	Confidence    float64
	ErrorText     string
}
