package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type GeminiProvider struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

// Gemini-specific request payload structure
type geminiRequest struct {
	Contents          []map[string]any `json:"contents"`
	SystemInstruction *map[string]any  `json:"systemInstruction,omitempty"`
	GenerationConfig  map[string]any   `json:"generationConfig,omitempty"`
}

// Gemini-specific response payload structure
type geminiResponse struct {
	Candidates []struct {
		Content struct {
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
	// 1. Build the Gemini-specific request payload
	payload := geminiRequest{
		Contents:         toGeminiContents(req.Messages),
		GenerationConfig: make(map[string]any),
	}
	if req.Temperature > 0 {
		payload.GenerationConfig["temperature"] = req.Temperature
	}
	if req.TopP > 0 {
		payload.GenerationConfig["topP"] = req.TopP
	}
	if req.MaxTokens > 0 {
		payload.GenerationConfig["maxOutputTokens"] = req.MaxTokens
	}
	if sys := firstSystemMessage(req.Messages); sys != "" {
		payload.SystemInstruction = &map[string]any{
			"role":  "system",
			"parts": []map[string]any{{"text": sys}},
		}
	}

	// 2. Prepare headers and call the generic helper
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", g.BaseURL, firstNonEmpty(req.Model, g.Model))
	headers := http.Header{
		"Content-Type":   []string{"application/json"},
		"x-goog-api-key": []string{g.APIKey},
	}

	wire, rawResp, err := httpRequest[geminiRequest, geminiResponse](ctx, g.Client, url, headers, payload)
	if err != nil {
		return nil, fmt.Errorf("gemini: http error: %w body: %s", err, string(rawResp))
	}

	// 3. Map the Gemini response to our standard LLMResponse
	out := &LLMResponse{
		Raw: json.RawMessage(rawResp),
		Usage: &Usage{
			PromptTokens:     wire.Usage.PromptTokenCount,
			CompletionTokens: wire.Usage.CandidatesTokenCount,
			TotalTokens:      wire.Usage.TotalTokenCount,
		},
	}
	if len(wire.Candidates) > 0 {
		first := wire.Candidates[0]
		var buf bytes.Buffer
		for _, p := range first.Content.Parts {
			buf.WriteString(p.Text) // Simplified text extraction
		}
		out.Text = buf.String()
		out.FinishReason = first.FinishReason
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
