package llm

import "encoding/json"

const (
	SchemaVersion = "hic.llm.offering_repair.v1"
	PromptVersion = "hic-offering-repair-2026-05-15"
	DefaultModel  = "gpt-5.4-mini"
)

type RepairInput struct {
	ArtifactID    int64
	ArtifactType  string
	NoticeSeq     string
	NoticeTitle   string
	OriginalFile  string
	SourceSpan    string
	RawText       string
	ContentJSON   json.RawMessage
	Confidence    float64
	MaxInputChars int
}

type Offering struct {
	ApplicationUnitLabel string   `json:"application_unit_label"`
	HousingName          string   `json:"housing_name"`
	UnitNo               *string  `json:"unit_no"`
	ExclusiveAreaM2      *float64 `json:"exclusive_area_m2"`
	SupplyCount          *int     `json:"supply_count"`
	DepositKRW           *int64   `json:"deposit_krw"`
	MonthlyRentKRW       *int64   `json:"monthly_rent_krw"`
	JeonseDepositKRW     *int64   `json:"jeonse_deposit_krw"`
	DormitoryFeeKRW      *int64   `json:"dormitory_fee_krw"`
	GenderRequirement    string   `json:"gender_requirement"`
	SourcePage           *int     `json:"source_page"`
	SourceSpan           string   `json:"source_span"`
	Confidence           float64  `json:"confidence"`
	RawEvidence          string   `json:"raw_evidence"`
}

type RepairOutput struct {
	SchemaVersion string     `json:"schema_version"`
	PromptVersion string     `json:"prompt_version"`
	Model         string     `json:"model"`
	InputHash     string     `json:"input_hash"`
	Confidence    float64    `json:"confidence"`
	SourceSpan    string     `json:"source_span"`
	Offerings     []Offering `json:"offerings"`
}

type AttemptRecord struct {
	ArtifactID    int64
	SchemaVersion string
	PromptVersion string
	Model         string
	InputHash     string
	OutputHash    string
	Status        string
	Confidence    float64
	SourceSpan    string
	RequestJSON   []byte
	ResponseJSON  []byte
	ErrorText     string
}
