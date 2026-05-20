package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultEndpoint = "https://api.openai.com/v1/responses"

type Client struct {
	APIKey     string
	Model      string
	Endpoint   string
	HTTPClient *http.Client
}

func (c Client) RepairOfferings(ctx context.Context, input RepairInput) (RepairOutput, AttemptRecord, error) {
	request, inputHash, err := BuildRepairRequest(input, c.Model)
	if err != nil {
		return RepairOutput{}, AttemptRecord{}, err
	}
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return RepairOutput{}, AttemptRecord{}, fmt.Errorf("marshal LLM request: %w", err)
	}

	model := c.Model
	if model == "" {
		model = DefaultModel
	}
	attempt := AttemptRecord{
		ArtifactID:    input.ArtifactID,
		SchemaVersion: SchemaVersion,
		PromptVersion: PromptVersion,
		Model:         model,
		InputHash:     inputHash,
		Status:        "failed",
		RequestJSON:   requestJSON,
	}

	if strings.TrimSpace(c.APIKey) == "" {
		attempt.ErrorText = "OPENAI_API_KEY is required"
		return RepairOutput{}, attempt, errors.New(attempt.ErrorText)
	}

	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(requestJSON))
	if err != nil {
		attempt.ErrorText = err.Error()
		return RepairOutput{}, attempt, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		attempt.ErrorText = err.Error()
		return RepairOutput{}, attempt, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		attempt.ErrorText = err.Error()
		return RepairOutput{}, attempt, err
	}
	attempt.ResponseJSON = body

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		attempt.ErrorText = fmt.Sprintf("OpenAI response status %d: %s", resp.StatusCode, string(body))
		return RepairOutput{}, attempt, errors.New(attempt.ErrorText)
	}

	text, err := responseText(body)
	if err != nil {
		attempt.ErrorText = err.Error()
		return RepairOutput{}, attempt, err
	}

	var parsed struct {
		Confidence float64    `json:"confidence"`
		SourceSpan string     `json:"source_span"`
		Offerings  []Offering `json:"offerings"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		attempt.ErrorText = fmt.Sprintf("parse structured LLM output: %v", err)
		return RepairOutput{}, attempt, errors.New(attempt.ErrorText)
	}

	output := RepairOutput{
		SchemaVersion: SchemaVersion,
		PromptVersion: PromptVersion,
		Model:         model,
		InputHash:     inputHash,
		Confidence:    parsed.Confidence,
		SourceSpan:    parsed.SourceSpan,
		Offerings:     parsed.Offerings,
	}
	attempt.OutputHash = outputHash(output)
	attempt.Status = "succeeded"
	attempt.Confidence = output.Confidence
	attempt.SourceSpan = output.SourceSpan
	return output, attempt, nil
}

func responseText(body []byte) (string, error) {
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return "", fmt.Errorf("parse OpenAI response JSON: %w", err)
	}

	if text, ok := root["output_text"].(string); ok && strings.TrimSpace(text) != "" {
		return text, nil
	}

	output, _ := root["output"].([]any)
	for _, item := range output {
		itemMap, _ := item.(map[string]any)
		content, _ := itemMap["content"].([]any)
		for _, contentItem := range content {
			contentMap, _ := contentItem.(map[string]any)
			if text, ok := contentMap["text"].(string); ok && strings.TrimSpace(text) != "" {
				return text, nil
			}
		}
	}
	return "", errors.New("OpenAI response did not contain output text")
}
