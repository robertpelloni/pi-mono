package config

import (
	"fmt"
	"os"

	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/spf13/viper"
)

type Config struct {
	Providers map[string]ProviderConfig `json:"providers"`
}

type ProviderConfig struct {
	BaseURL        *string                  `json:"baseUrl,omitempty"`
	APIKey         *string                  `json:"apiKey,omitempty"`
	API            *string                  `json:"api,omitempty"`
	Headers        map[string]string        `json:"headers,omitempty"`
	Models         []ai.ModelInfo           `json:"models,omitempty"`
	ModelOverrides map[string]ModelOverride `json:"modelOverrides,omitempty"`
}

type ModelOverride struct {
	Name          *string                     `json:"name,omitempty"`
	Reasoning     *bool                       `json:"reasoning,omitempty"`
	ContextWindow *int                        `json:"contextWindow,omitempty"`
	MaxTokens     *int                        `json:"maxTokens,omitempty"`
	Input         []string                    `json:"input,omitempty"`
	Cost          *ModelCostOverride          `json:"cost,omitempty"`
	Headers       map[string]string           `json:"headers,omitempty"`
	Compat        *ai.OpenAICompletionsCompat `json:"compat,omitempty"`
}

type ModelCostOverride struct {
	Input      *float64 `json:"input,omitempty"`
	Output     *float64 `json:"output,omitempty"`
	CacheRead  *float64 `json:"cacheRead,omitempty"`
	CacheWrite *float64 `json:"cacheWrite,omitempty"`
}

var v *viper.Viper

func InitConfig(configPath string) (*Config, error) {
	v = viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("json")

	if err := v.ReadInConfig(); err != nil {
		if os.IsNotExist(err) {
			return &Config{Providers: make(map[string]ProviderConfig)}, nil
		}
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &c, nil
}

// MergeConfig merges the loaded configuration into the global AI model registry
func MergeConfig(c *Config) {
	if c == nil {
		return
	}

	for providerName, providerConfig := range c.Providers {
		provider := ai.Provider(providerName)

		// Apply newly defined models
		for _, model := range providerConfig.Models {
			// Ensure essential fields are set based on the provider config overrides
			if providerConfig.BaseURL != nil {
				model.BaseURL = *providerConfig.BaseURL
			}
			if providerConfig.API != nil {
				model.API = ai.Api(*providerConfig.API)
			}
			if model.Provider == "" {
				model.Provider = provider
			}
			ai.RegisterModel(model)
		}

		// Apply overrides to existing models
		for modelID, override := range providerConfig.ModelOverrides {
			existingModel, ok := ai.GetModel(provider, modelID)
			if ok {
				if override.Name != nil {
					existingModel.Name = *override.Name
				}
				if override.Reasoning != nil {
					existingModel.Reasoning = *override.Reasoning
				}
				if override.ContextWindow != nil {
					existingModel.ContextWindow = *override.ContextWindow
				}
				if override.MaxTokens != nil {
					existingModel.MaxTokens = *override.MaxTokens
				}
				if override.Cost != nil {
					if override.Cost.Input != nil {
						existingModel.Cost.Input = *override.Cost.Input
					}
					if override.Cost.Output != nil {
						existingModel.Cost.Output = *override.Cost.Output
					}
					if override.Cost.CacheRead != nil {
						existingModel.Cost.CacheRead = *override.Cost.CacheRead
					}
					if override.Cost.CacheWrite != nil {
						existingModel.Cost.CacheWrite = *override.Cost.CacheWrite
					}
				}
				// Save back to registry
				ai.RegisterModel(*existingModel)
			}
		}

		// Apply provider-wide overrides to all models belonging to this provider
		if providerConfig.BaseURL != nil {
			models := ai.GetModels(provider)
			for _, model := range models {
				model.BaseURL = *providerConfig.BaseURL
				ai.RegisterModel(model)
			}
		}
	}
}
