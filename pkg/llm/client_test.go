package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildRepairRequest_JSONSchemaResponses요청을생성한다(t *testing.T) {
	request, inputHash, err := BuildRepairRequest(RepairInput{
		ArtifactID:   10,
		ArtifactType: "pdf_text",
		NoticeTitle:  "입주자 모집공고",
		SourceSpan:   "pdf://sample#page=3",
		RawText:      "공급호수 2",
	}, "gpt-test")
	if err != nil {
		t.Fatalf("BuildRepairRequest() error = %v", err)
	}
	if inputHash == "" {
		t.Fatalf("inputHash is empty")
	}
	if request["model"] != "gpt-test" {
		t.Fatalf("model = %v", request["model"])
	}
	text := request["text"].(map[string]any)
	format := text["format"].(map[string]any)
	if format["type"] != "json_schema" || format["strict"] != true {
		t.Fatalf("format = %+v", format)
	}
	schema := format["schema"].(map[string]any)
	if schema["additionalProperties"] != false {
		t.Fatalf("schema should disallow additional properties: %+v", schema)
	}
}

func TestClient_RepairOfferings_ResponsesAPI결과를파싱한다(t *testing.T) {
	var gotAuthorization string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthorization = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("Decode request error = %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "resp_test",
			"output": [{
				"type": "message",
				"content": [{
					"type": "output_text",
					"text": "{\"confidence\":0.82,\"source_span\":\"pdf://sample#page=3\",\"offerings\":[{\"application_unit_label\":\"청담르엘 49 일반\",\"housing_name\":\"청담르엘\",\"unit_no\":null,\"exclusive_area_m2\":49,\"supply_count\":15,\"deposit_krw\":null,\"monthly_rent_krw\":null,\"jeonse_deposit_krw\":772980000,\"dormitory_fee_krw\":null,\"gender_requirement\":\"\",\"source_page\":3,\"source_span\":\"pdf://sample#page=3&row=1\",\"confidence\":0.82,\"raw_evidence\":\"청담르엘 49 15 772,980\"}]}"
				}]
			}]
		}`))
	}))
	defer server.Close()

	client := Client{APIKey: "sk-test", Model: "gpt-test", Endpoint: server.URL, HTTPClient: server.Client()}
	output, attempt, err := client.RepairOfferings(context.Background(), RepairInput{
		ArtifactID:   10,
		ArtifactType: "pdf_text",
		NoticeTitle:  "입주자 모집공고",
		SourceSpan:   "pdf://sample#page=3",
		RawText:      "청담르엘 49 15 772,980",
	})
	if err != nil {
		t.Fatalf("RepairOfferings() error = %v", err)
	}
	if gotAuthorization != "Bearer sk-test" {
		t.Fatalf("Authorization = %q", gotAuthorization)
	}
	if gotBody["model"] != "gpt-test" {
		t.Fatalf("request model = %v", gotBody["model"])
	}
	if attempt.Status != "succeeded" || attempt.InputHash == "" || attempt.OutputHash == "" {
		t.Fatalf("attempt = %+v", attempt)
	}
	if output.SchemaVersion != SchemaVersion || output.PromptVersion != PromptVersion {
		t.Fatalf("metadata = %+v", output)
	}
	if len(output.Offerings) != 1 || output.Offerings[0].HousingName != "청담르엘" {
		t.Fatalf("offerings = %+v", output.Offerings)
	}
}

func TestClient_RepairOfferings_APIKey를요구한다(t *testing.T) {
	client := Client{Model: "gpt-test"}
	_, attempt, err := client.RepairOfferings(context.Background(), RepairInput{ArtifactID: 1})
	if err == nil {
		t.Fatalf("RepairOfferings() error = nil")
	}
	if attempt.Status != "failed" || !strings.Contains(attempt.ErrorText, "OPENAI_API_KEY") {
		t.Fatalf("attempt = %+v", attempt)
	}
}
