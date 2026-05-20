package llm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

func BuildRepairRequest(input RepairInput, model string) (map[string]any, string, error) {
	if model == "" {
		model = DefaultModel
	}
	payload := buildInputPayload(input)
	hash := sha256.Sum256([]byte(payload))
	inputHash := hex.EncodeToString(hash[:])

	request := map[string]any{
		"model": model,
		"input": []map[string]any{
			{
				"role":    "system",
				"content": "You extract public housing application-selectable offerings from SH notice artifacts. Return JSON only. Do not invent rows. Every offering must include source evidence from the provided artifact.",
			},
			{
				"role":    "user",
				"content": payload,
			},
		},
		"text": map[string]any{
			"format": map[string]any{
				"type":        "json_schema",
				"name":        "hic_offering_repair",
				"description": "HIC LLM repair output for application-selectable public housing offerings.",
				"strict":      true,
				"schema":      repairJSONSchema(),
			},
		},
	}
	return request, inputHash, nil
}

func buildInputPayload(input RepairInput) string {
	maxChars := input.MaxInputChars
	if maxChars <= 0 {
		maxChars = 12000
	}

	rawText := strings.TrimSpace(input.RawText)
	if len([]rune(rawText)) > maxChars {
		runes := []rune(rawText)
		rawText = string(runes[:maxChars]) + "\n[TRUNCATED]"
	}
	contentJSON := strings.TrimSpace(string(input.ContentJSON))
	if contentJSON == "" {
		contentJSON = "{}"
	}

	parts := []string{
		"Task: Extract HIC Offering records from this artifact.",
		"Definitions:",
		"- Offering means an application-selectable unit. It may be one concrete room/unit or one grouped row with supply_count.",
		"- If exact unit numbers are not published, unit_no must be null and supply_count should represent the row's number of supplied units.",
		"- Exclude winner announcements, contract guides, applicant lists, Q&A, forms, and generic instructions.",
		"- Use null for unknown numeric fields. Do not guess money or counts.",
		"",
		"Artifact metadata:",
		"artifact_id: " + int64String(input.ArtifactID),
		"artifact_type: " + input.ArtifactType,
		"notice_seq: " + input.NoticeSeq,
		"notice_title: " + input.NoticeTitle,
		"original_file: " + input.OriginalFile,
		"source_span: " + input.SourceSpan,
		"",
		"content_json:",
		contentJSON,
		"",
		"raw_text:",
		rawText,
	}
	return strings.Join(parts, "\n")
}

func repairJSONSchema() map[string]any {
	nullableString := []any{"string", "null"}
	nullableNumber := []any{"number", "null"}
	nullableInteger := []any{"integer", "null"}

	offering := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required": []string{
			"application_unit_label",
			"housing_name",
			"unit_no",
			"exclusive_area_m2",
			"supply_count",
			"deposit_krw",
			"monthly_rent_krw",
			"jeonse_deposit_krw",
			"dormitory_fee_krw",
			"gender_requirement",
			"source_page",
			"source_span",
			"confidence",
			"raw_evidence",
		},
		"properties": map[string]any{
			"application_unit_label": map[string]any{"type": "string"},
			"housing_name":           map[string]any{"type": "string"},
			"unit_no":                map[string]any{"type": nullableString},
			"exclusive_area_m2":      map[string]any{"type": nullableNumber},
			"supply_count":           map[string]any{"type": nullableInteger},
			"deposit_krw":            map[string]any{"type": nullableInteger},
			"monthly_rent_krw":       map[string]any{"type": nullableInteger},
			"jeonse_deposit_krw":     map[string]any{"type": nullableInteger},
			"dormitory_fee_krw":      map[string]any{"type": nullableInteger},
			"gender_requirement":     map[string]any{"type": "string"},
			"source_page":            map[string]any{"type": nullableInteger},
			"source_span":            map[string]any{"type": "string"},
			"confidence":             map[string]any{"type": "number", "minimum": 0, "maximum": 1},
			"raw_evidence":           map[string]any{"type": "string"},
		},
	}

	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"confidence", "source_span", "offerings"},
		"properties": map[string]any{
			"confidence":  map[string]any{"type": "number", "minimum": 0, "maximum": 1},
			"source_span": map[string]any{"type": "string"},
			"offerings": map[string]any{
				"type":  "array",
				"items": offering,
			},
		},
	}
}

func outputHash(output RepairOutput) string {
	b, _ := json.Marshal(output)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func int64String(value int64) string {
	b, _ := json.Marshal(value)
	return string(b)
}
