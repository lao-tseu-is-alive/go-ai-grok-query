package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type GeminiProvider struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

func newGeminiAdapter(cfg ProviderConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("gemini: missing API key")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("gemini: missing model")
	}
	base := cfg.BaseURL
	if base == "" {
		base = "https://generativelanguage.googleapis.com"
	}
	return &GeminiProvider{
		BaseURL: base,
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		Client:  &http.Client{Timeout: 0},
	}, nil
}

func (g *GeminiProvider) Query(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", g.BaseURL, firstNonEmpty(req.Model, g.Model))

	contents := toGeminiContents(req.Messages)
	payload := map[string]any{
		"contents": contents,
		// Map temperature, topP, maxOutputTokens where provided
		"generationConfig": map[string]any{},
	}
	gen := payload["generationConfig"].(map[string]any)
	if req.Temperature > 0 {
		gen["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		gen["topP"] = req.TopP
	}
	if req.MaxTokens > 0 {
		gen["maxOutputTokens"] = req.MaxTokens
	}
	// If a system message exists, send as systemInstruction (Gemini-native)
	if sys := firstSystemMessage(req.Messages); sys != "" {
		payload["systemInstruction"] = map[string]any{
			"role":  "system",
			"parts": []map[string]any{{"text": sys}},
		}
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("gemini: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", g.APIKey)

	resp, err := g.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: do request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gemini: read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gemini: http %d: %s", resp.StatusCode, string(respBody))
	}

	var wire struct {
		Candidates []struct {
			Content struct {
				Role  string `json:"role"`
				Parts []struct {
					Text string `json:"text,omitempty"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason,omitempty"`
		} `json:"candidates"`
		Usage struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(respBody, &wire); err != nil {
		return nil, fmt.Errorf("gemini: unmarshal: %w", err)
	}

	out := &LLMResponse{Raw: json.RawMessage(respBody)}
	if len(wire.Candidates) > 0 {
		first := wire.Candidates[0]
		var buf bytes.Buffer
		for _, p := range first.Content.Parts {
			if p.Text != "" {
				if buf.Len() > 0 {
					buf.WriteByte('\n')
				}
				buf.WriteString(p.Text)
			}
		}
		out.Text = buf.String()
		out.FinishReason = first.FinishReason
	}
	out.Usage = &Usage{
		PromptTokens:     wire.Usage.PromptTokenCount,
		CompletionTokens: wire.Usage.CandidatesTokenCount,
		TotalTokens:      wire.Usage.TotalTokenCount,
	}
	return out, nil

}

func (g *GeminiProvider) Stream(ctx context.Context, req *LLMRequest, onDelta func(Delta)) (*LLMResponse, error) {
	// TODO: implement /streamGenerateContent with incremental candidate chunks.
	return nil, fmt.Errorf("gemini: streaming not implemented")
}

func (g *GeminiProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	// TODO: implement GET /v1beta/models for listing and feature flags
	return nil, fmt.Errorf("gemini: ListModels not implemented")
}

// Helpers

func toGeminiContents(msgs []LLMMessage) []map[string]any {
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		if m.Role == RoleSystem {
			// handled via systemInstruction
			continue
		}
		out = append(out, map[string]any{
			"role": m.Role,
			"parts": []map[string]any{
				{"text": m.Content},
			},
		})
	}
	return out
}

func firstSystemMessage(msgs []LLMMessage) string {
	for _, m := range msgs {
		if m.Role == RoleSystem && m.Content != "" {
			return m.Content
		}
	}
	return ""
}
