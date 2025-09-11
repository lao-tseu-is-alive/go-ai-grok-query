package llm

import (
	"encoding/json"
	"os"
)

// ModelOverride defines optional fields to override the provider's defaults.
// Using pointers allows us to distinguish between a field being explicitly set to `false`
// and a field not being set at all.
type ModelOverride struct {
	ContextSize        *int  `json:"context_size,omitempty"`
	SupportsTools      *bool `json:"supports_tools,omitempty"`
	SupportsThinking   *bool `json:"supports_thinking,omitempty"`
	SupportsInputImage *bool `json:"supports_input_image,omitempty"`
	SupportsStreaming  *bool `json:"supports_streaming,omitempty"`
	SupportsJSONMode   *bool `json:"supports_json_mode,omitempty"`
	SupportsStructured *bool `json:"supports_structured,omitempty"`
}

// ProviderModelsInfo holds the model catalog for a single provider.
// It uses the existing llm.ModelInfo struct.
type ProviderModelsInfo struct {
	Models          map[string]ModelOverride `json:"models"`
	Defaults        ModelInfo                `json:"defaults"`
	ExcludePatterns []string                 `json:"exclude_patterns"`
}

// ModelCatalog is the top-level structure for the entire models.json file.
type ModelCatalog struct {
	Version   int                           `json:"version"`
	Providers map[string]ProviderModelsInfo `json:"providers"`
}

// LoadModelCatalog reads and parses the models.json file from the given path.
func LoadModelCatalog(filePath string) (*ModelCatalog, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var catalog ModelCatalog
	if err := json.Unmarshal(file, &catalog); err != nil {
		return nil, err
	}

	return &catalog, nil
}

// MergeModelInfo combines a default ModelInfo with specific overrides.
// It starts with the default values and replaces them with any non-zero or true values from the overrides.
func MergeModelInfo(defaults ModelInfo, overrides ModelOverride) ModelInfo {
	merged := defaults

	if overrides.ContextSize != nil {
		merged.ContextSize = *overrides.ContextSize
	}
	if overrides.SupportsTools != nil {
		merged.SupportsTools = *overrides.SupportsTools
	}
	if overrides.SupportsThinking != nil {
		merged.SupportsThinking = *overrides.SupportsThinking
	}
	if overrides.SupportsInputImage != nil {
		merged.SupportsInputImage = *overrides.SupportsInputImage
	}
	if overrides.SupportsStreaming != nil {
		merged.SupportsStreaming = *overrides.SupportsStreaming
	}
	if overrides.SupportsJSONMode != nil {
		merged.SupportsJSONMode = *overrides.SupportsJSONMode
	}
	if overrides.SupportsStructured != nil {
		merged.SupportsStructured = *overrides.SupportsStructured
	}

	return merged
}
