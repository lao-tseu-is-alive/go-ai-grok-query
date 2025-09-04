package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"time"
)

// GeminiProvider implements the Provider interface for Google's Gemini models.
type GeminiProvider struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

// geminiRequest represents the request payload for Gemini's generateContent API.
type geminiRequest struct {
	Contents          []map[string]any `json:"contents"`
	SystemInstruction *map[string]any  `json:"systemInstruction,omitempty"`
	GenerationConfig  map[string]any   `json:"generationConfig,omitempty"`
}

// geminiResponse represents the response payload from Gemini's generateContent API.
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

// NewGeminiAdapter creates a new GeminiProvider from config.
func NewGeminiAdapter(cfg ProviderConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("gemini: API key required") // Shorter, error-based
	}
	if cfg.Model == "" {
		return nil, errors.New("gemini: model required")
	}
	baseURL := FirstNonEmpty(cfg.BaseURL, "https://generativelanguage.googleapis.com")
	return &GeminiProvider{
		BaseURL: baseURL,
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		Client:  &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (g *GeminiProvider) Query(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}

	payload := geminiRequest{
		Contents:         ToGeminiContents(req.Messages), // Exported helper
		GenerationConfig: map[string]any{},
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
	if sys := FirstSystemMessage(req.Messages); sys != "" {
		payload.SystemInstruction = &map[string]any{
			"role":  "system",
			"parts": []map[string]any{{"text": sys}},
		}
	}

	url := g.BaseURL + "/v1beta/models/" + path.Join(FirstNonEmpty(req.Model, g.Model), ":generateContent") // Safer path join
	headers := http.Header{
		"Content-Type":   []string{"application/json"},
		"x-goog-api-key": []string{g.APIKey},
	}

	responseData, rawResp, err := HttpRequest[geminiRequest, geminiResponse](ctx, g.Client, url, headers, payload)
	if err != nil {
		return nil, fmt.Errorf("gemini request failed: %w (raw body: %s)", err, string(rawResp))
	}

	llmResp := &LLMResponse{
		Raw: json.RawMessage(rawResp),
		Usage: &Usage{
			PromptTokens:     responseData.Usage.PromptTokenCount,
			CompletionTokens: responseData.Usage.CandidatesTokenCount,
			TotalTokens:      responseData.Usage.TotalTokenCount,
		},
	}
	if len(responseData.Candidates) > 0 {
		var buf bytes.Buffer
		for _, part := range responseData.Candidates[0].Content.Parts {
			buf.WriteString(part.Text)
		}
		llmResp.Text = buf.String()
		llmResp.FinishReason = responseData.Candidates[0].FinishReason
	}

	return llmResp, nil
}

func (g *GeminiProvider) Stream(ctx context.Context, req *LLMRequest, onDelta func(Delta)) (*LLMResponse, error) {
	return nil, errors.New("gemini streaming not implemented")
}

func (g *GeminiProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	return nil, errors.New("gemini list models not implemented")
}

// ToGeminiContents converts LLM messages to Gemini's content format.
func ToGeminiContents(msgs []LLMMessage) []map[string]any {
	out := make([]map[string]any, 0, len(msgs))
	for _, msg := range msgs {
		if msg.Role == RoleSystem {
			continue
		}
		out = append(out, map[string]any{
			"role":  msg.Role,
			"parts": []map[string]any{{"text": msg.Content}},
		})
	}
	return out
}

// FirstSystemMessage finds the first system message content.
func FirstSystemMessage(msgs []LLMMessage) string {
	for _, msg := range msgs {
		if msg.Role == RoleSystem && msg.Content != "" {
			return msg.Content
		}
	}
	return ""
}
