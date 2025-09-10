package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
)

// GeminiProvider implements the Provider interface for Google's Gemini models.
type GeminiProvider struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
	l       golog.MyLogger
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
func NewGeminiAdapter(cfg ProviderConfig, l golog.MyLogger) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("gemini: API key required") // Shorter, error-based
	}
	if cfg.Model == "" {
		return nil, errors.New("gemini: model required")
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("gemini: missing baseURl")
	}
	return &GeminiProvider{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		Client:  &http.Client{Timeout: 30 * time.Second},
		l:       l,
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

	g.l.Debug("about to send request to %s", g.BaseURL)
	responseData, rawResp, err := HttpRequest[geminiRequest, geminiResponse](ctx, g.Client, url, headers, payload, g.l)
	if err != nil {
		g.l.Warn("got error during HttpRequest: %q", err)
		return nil, fmt.Errorf("gemini request failed: %w (raw body: %s)", err, string(rawResp))
	}
	g.l.Debug("successful HttpRequest, rawbody: %s", string(rawResp))

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

func (g *GeminiProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	url := g.BaseURL + "/v1beta/models"
	headers := http.Header{
		"x-goog-api-key": []string{g.APIKey},
	}

	type geminiModelsResponse struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	resp, err := httpGetRequest[geminiModelsResponse](ctx, g.Client, url, headers, g.l)
	if err != nil {
		return nil, fmt.Errorf("failed to list gemini models: %w", err)
	}

	modelInfos := make([]ModelInfo, len(resp.Models))
	for i, model := range resp.Models {
		modelInfos[i] = ModelInfo{Name: model.Name}
	}

	return modelInfos, nil
}

func (g *GeminiProvider) Stream(ctx context.Context, req *LLMRequest, onDelta func(Delta)) (*LLMResponse, error) {
	// 1. Validate inputs and build the request payload
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	if onDelta == nil {
		return nil, errors.New("onDelta callback cannot be nil for streaming")
	}

	payload := geminiRequest{
		Contents:         ToGeminiContents(req.Messages),
		GenerationConfig: map[string]any{},
	}
	if req.Temperature > 0 {
		payload.GenerationConfig["temperature"] = req.Temperature
	}
	if sys := FirstSystemMessage(req.Messages); sys != "" {
		payload.SystemInstruction = &map[string]any{
			"role":  "system",
			"parts": []map[string]any{{"text": sys}},
		}
	}

	// 2. Prepare and send the HTTP request
	modelName := FirstNonEmpty(req.Model, g.Model)
	url := g.BaseURL + "/v1beta/models/" + path.Join(modelName, ":streamGenerateContent")
	headers := http.Header{
		"Content-Type":   []string{"application/json"},
		"x-goog-api-key": []string{g.APIKey},
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gemini stream request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini stream request: %w", err)
	}
	httpReq.Header = headers
	g.l.Debug("Gemini stream request sent to URL: %s", url)
	resp, err := g.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send gemini stream request: %w", err)
	}
	defer resp.Body.Close()
	g.l.Debug("Gemini stream response status: %s", resp.Status)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini stream returned non-200 status: %d %s", resp.StatusCode, string(body))
	}

	// 3.  Process the response as a streaming JSON array, not as SSE.
	decoder := json.NewDecoder(resp.Body)
	finalResponse := &LLMResponse{}
	fullText := &strings.Builder{}

	// The entire response is a single JSON array. We first must read the opening token '['.
	t, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to read opening token of JSON array: %w", err)
	}
	if t != json.Delim('[') {
		return nil, fmt.Errorf("expected '[' at start of stream, but got %v", t)
	}
	g.l.Debug("Successfully found opening '[' of the JSON array.")

	// Now, we loop through the array, decoding one full JSON object at a time.
	for decoder.More() {
		var chunk geminiResponse
		if err := decoder.Decode(&chunk); err != nil {
			g.l.Warn("Failed to decode gemini object from stream: %v", err)
			continue
		}
		g.l.Debug("Successfully decoded one object from the stream array.")

		// The logic for processing the chunk is the same as before.
		if len(chunk.Candidates) > 0 {
			candidate := chunk.Candidates[0]
			if len(candidate.Content.Parts) > 0 {
				textDelta := candidate.Content.Parts[0].Text
				if textDelta != "" {
					g.l.Debug("Extracted delta: '%s'", textDelta)
					fullText.WriteString(textDelta)
					onDelta(Delta{Text: textDelta})
				}
			}
			if candidate.FinishReason != "" {
				finalResponse.FinishReason = candidate.FinishReason
			}
		}
		if chunk.Usage.TotalTokenCount > 0 {
			finalResponse.Usage = &Usage{
				PromptTokens:     chunk.Usage.PromptTokenCount,
				CompletionTokens: chunk.Usage.CandidatesTokenCount,
				TotalTokens:      chunk.Usage.TotalTokenCount,
			}
		}
	}

	g.l.Debug("Finished processing Gemini stream.")
	onDelta(Delta{Done: true, FinishReason: finalResponse.FinishReason})
	finalResponse.Text = fullText.String()
	return finalResponse, nil
}
