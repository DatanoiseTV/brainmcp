package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// Config holds application configuration from ~/.brainmcp/config.json
type Config struct {
	EmbeddingProvider string         `json:"embedding_provider,omitempty"` // "gemini" or "lmstudio"
	Qdrant            QdrantConfig   `json:"qdrant,omitempty"`
	Gemini            GeminiConfig   `json:"gemini,omitempty"`
	LMStudio          LMStudioConfig `json:"lmstudio,omitempty"`
}

// QdrantConfig holds Qdrant connection settings.
type QdrantConfig struct {
	Host            string `json:"host,omitempty"`
	Port            int    `json:"port,omitempty"`
	APIKey          string `json:"api_key,omitempty"`
	UseTLS          bool   `json:"use_tls"`
	VectorDimension int    `json:"vector_dimension,omitempty"`
}

// GeminiConfig holds Gemini model settings.
type GeminiConfig struct {
	APIKey         string `json:"api_key,omitempty"`
	EmbeddingModel string `json:"embedding_model,omitempty"`
	LLMModel       string `json:"llm_model,omitempty"`
}

// LMStudioConfig holds LM Studio connection settings.
type LMStudioConfig struct {
	BaseURL        string `json:"base_url,omitempty"`
	EmbeddingModel string `json:"embedding_model,omitempty"`
}

// LoadConfig reads configuration from ~/.brainmcp/config.json
func LoadConfig(logger *log.Logger) (*Config, error) {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".brainmcp", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist, return empty config (use defaults)
			logger.Printf("Config file not found at %s, using defaults and environment variables", configPath)
			cfg := &Config{
				Qdrant: QdrantConfig{UseTLS: true, VectorDimension: 768},
				Gemini: GeminiConfig{},
			}
			// Load from environment variables if file is missing
			if host := os.Getenv("QDRANT_HOST"); host != "" {
				cfg.Qdrant.Host = host
			}
			if portStr := os.Getenv("QDRANT_PORT"); portStr != "" {
				var p int
				if _, err := fmt.Sscanf(portStr, "%d", &p); err == nil {
					cfg.Qdrant.Port = p
				}
			}
			if apiKey := os.Getenv("QDRANT_API_KEY"); apiKey != "" {
				cfg.Qdrant.APIKey = apiKey
			}
			if tlsStr := os.Getenv("QDRANT_USE_TLS"); tlsStr != "" {
				cfg.Qdrant.UseTLS = tlsStr == "1" || tlsStr == "true"
			}
			if geminiKey := os.Getenv("GEMINI_API_KEY"); geminiKey != "" {
				cfg.Gemini.APIKey = geminiKey
			}
			if provider := os.Getenv("EMBEDDING_PROVIDER"); provider != "" {
				cfg.EmbeddingProvider = provider
			}
			if baseURL := os.Getenv("LMSTUDIO_BASE_URL"); baseURL != "" {
				cfg.LMStudio.BaseURL = baseURL
			}
			if model := os.Getenv("LMSTUDIO_EMBEDDING_MODEL"); model != "" {
				cfg.LMStudio.EmbeddingModel = model
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config.json: %w", err)
	}

	// Override with environment variables if present
	if host := os.Getenv("QDRANT_HOST"); host != "" {
		cfg.Qdrant.Host = host
	}
	if portStr := os.Getenv("QDRANT_PORT"); portStr != "" {
		var p int
		if _, err := fmt.Sscanf(portStr, "%d", &p); err == nil {
			cfg.Qdrant.Port = p
		}
	}
	if apiKey := os.Getenv("QDRANT_API_KEY"); apiKey != "" {
		cfg.Qdrant.APIKey = apiKey
	}
	if tlsStr := os.Getenv("QDRANT_USE_TLS"); tlsStr != "" {
		cfg.Qdrant.UseTLS = tlsStr == "1" || tlsStr == "true"
	}

	if provider := os.Getenv("EMBEDDING_PROVIDER"); provider != "" {
		cfg.EmbeddingProvider = provider
	}
	if baseURL := os.Getenv("LMSTUDIO_BASE_URL"); baseURL != "" {
		cfg.LMStudio.BaseURL = baseURL
	}
	if model := os.Getenv("LMSTUDIO_EMBEDDING_MODEL"); model != "" {
		cfg.LMStudio.EmbeddingModel = model
	}

	if geminiKey := os.Getenv("GEMINI_API_KEY"); geminiKey != "" {
		cfg.Gemini.APIKey = geminiKey
	}
	if embModel := os.Getenv("GEMINI_EMBEDDING_MODEL"); embModel != "" {
		cfg.Gemini.EmbeddingModel = embModel
	}
	if llmModel := os.Getenv("GEMINI_LLM_MODEL"); llmModel != "" {
		cfg.Gemini.LLMModel = llmModel
	}

	// Set defaults
	if cfg.Qdrant.VectorDimension == 0 {
		cfg.Qdrant.VectorDimension = 768 // Default for Gemini embeddings
	}

	// Default provider if not set
	if cfg.EmbeddingProvider == "" {
		cfg.EmbeddingProvider = "gemini"
	}

	// Set LM Studio defaults if chosen
	if cfg.EmbeddingProvider == "lmstudio" {
		if cfg.LMStudio.BaseURL == "" {
			cfg.LMStudio.BaseURL = "http://localhost:1234/v1"
		}
		if cfg.LMStudio.EmbeddingModel == "" {
			cfg.LMStudio.EmbeddingModel = "nomic-embed-text-v1.5"
		}
	}

	if !cfg.Qdrant.UseTLS && cfg.Qdrant.Port == 0 {
		// If UseTLS not explicitly set, default to true
		cfg.Qdrant.UseTLS = true
	}

	logger.Printf("Loaded config from %s", configPath)
	return &cfg, nil
}

// SaveConfig writes configuration to ~/.brainmcp/config.json
func SaveConfig(cfg *Config, logger *log.Logger) error {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	brainDir := filepath.Join(homeDir, ".brainmcp")
	if err := os.MkdirAll(brainDir, 0755); err != nil {
		return fmt.Errorf("failed to create .brainmcp directory: %w", err)
	}

	configPath := filepath.Join(brainDir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config.json: %w", err)
	}

	logger.Printf("Saved config to %s", configPath)
	return nil
}
