package ai

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	cachedVertexADCCredentialsExists bool
	vertexADCOnce                    sync.Once
)

// hasVertexADCCredentials checks if Google Application Default Credentials exist.
func hasVertexADCCredentials() bool {
	vertexADCOnce.Do(func() {
		// Check GOOGLE_APPLICATION_CREDENTIALS env var first
		gacPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		exists := false
		if gacPath != "" {
			if _, err := os.Stat(gacPath); err == nil {
				exists = true
			}
		} else {
			// Fall back to default ADC path
			homeDir, err := os.UserHomeDir()
			if err == nil {
				adcPath := filepath.Join(homeDir, ".config", "gcloud", "application_default_credentials.json")
				if _, err := os.Stat(adcPath); err == nil {
					exists = true
				}
			}
		}
		cachedVertexADCCredentialsExists = exists
	})
	return cachedVertexADCCredentialsExists
}

// GetEnvAPIKey gets the API key for a provider from known environment variables.
// Returns an empty string if no key is found.
// Will return "<authenticated>" for providers using alternative authentication (like AWS/GCP ADCs).
func GetEnvAPIKey(provider Provider) string {
	switch provider {
	case ProviderGithubCopilot:
		if token := os.Getenv("COPILOT_GITHUB_TOKEN"); token != "" {
			return token
		}
		if token := os.Getenv("GH_TOKEN"); token != "" {
			return token
		}
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			return token
		}
		return ""

	case ProviderAnthropic:
		// ANTHROPIC_OAUTH_TOKEN takes precedence over ANTHROPIC_API_KEY
		if token := os.Getenv("ANTHROPIC_OAUTH_TOKEN"); token != "" {
			return token
		}
		if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
			return key
		}
		return ""

	case ProviderVertex:
		if key := os.Getenv("GOOGLE_CLOUD_API_KEY"); key != "" {
			return key
		}

		hasCredentials := hasVertexADCCredentials()
		hasProject := os.Getenv("GOOGLE_CLOUD_PROJECT") != "" || os.Getenv("GCLOUD_PROJECT") != ""
		hasLocation := os.Getenv("GOOGLE_CLOUD_LOCATION") != ""

		if hasCredentials && hasProject && hasLocation {
			return "<authenticated>"
		}
		return ""

	case ProviderAmazonBedrock:
		if os.Getenv("AWS_PROFILE") != "" ||
			(os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "") ||
			os.Getenv("AWS_BEARER_TOKEN_BEDROCK") != "" ||
			os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI") != "" ||
			os.Getenv("AWS_CONTAINER_CREDENTIALS_FULL_URI") != "" ||
			os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE") != "" {
			return "<authenticated>"
		}
		return ""
	}

	envMap := map[Provider]string{
		ProviderOpenAI:        "OPENAI_API_KEY",
		ProviderAzureOpenAI:   "AZURE_OPENAI_API_KEY",
		ProviderGoogle:        "GEMINI_API_KEY",
		ProviderGroq:          "GROQ_API_KEY",
		ProviderCerebras:      "CEREBRAS_API_KEY",
		ProviderXAI:           "XAI_API_KEY",
		ProviderOpenRouter:    "OPENROUTER_API_KEY",
		ProviderVercelGateway: "AI_GATEWAY_API_KEY",
		ProviderZAI:           "ZAI_API_KEY",
		ProviderMistral:       "MISTRAL_API_KEY",
		ProviderMinimax:       "MINIMAX_API_KEY",
		ProviderMinimaxCN:     "MINIMAX_CN_API_KEY",
		ProviderHuggingFace:   "HF_TOKEN",
		ProviderOpenCode:      "OPENCODE_API_KEY",
		ProviderOpenCodeGo:    "OPENCODE_API_KEY",
		ProviderKimiCoding:    "KIMI_API_KEY",
	}

	if envVar, ok := envMap[provider]; ok {
		return os.Getenv(envVar)
	}

	return ""
}
