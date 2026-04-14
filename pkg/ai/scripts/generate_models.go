package main

import (
	"log"
)

// ModelsDevModel maps the JSON structure returned by the models.dev API.
type ModelsDevModel struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ToolCall  *bool  `json:"tool_call,omitempty"`
	Reasoning *bool  `json:"reasoning,omitempty"`
	Limit     *struct {
		Context *int `json:"context,omitempty"`
		Output  *int `json:"output,omitempty"`
	} `json:"limit,omitempty"`
	Cost *struct {
		Input      *float64 `json:"input,omitempty"`
		Output     *float64 `json:"output,omitempty"`
		CacheRead  *float64 `json:"cache_read,omitempty"`
		CacheWrite *float64 `json:"cache_write,omitempty"`
	} `json:"cost,omitempty"`
	Provider string `json:"provider"`
}

// OpenRouterModel maps the JSON structure returned by the OpenRouter API.
type OpenRouterModel struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Pricing *struct {
		Prompt     *string `json:"prompt,omitempty"`
		Completion *string `json:"completion,omitempty"`
	} `json:"pricing,omitempty"`
	ContextLength *int `json:"context_length,omitempty"`
	TopProvider   *struct {
		MaxCompletionTokens *int `json:"max_completion_tokens,omitempty"`
	} `json:"top_provider,omitempty"`
}

// VercelGatewayModel maps the JSON structure returned by the Vercel AI Gateway API.
type VercelGatewayModel struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
}

// This script simulates the behavior of the TypeScript packages/ai/scripts/generate-models.ts.
// It fetches available models from external APIs, filters for tool-calling capabilities,
// normalizes them into the ModelInfo struct, and generates a static output (e.g. models.json or models_generated.go).
func main() {
	log.Println("Starting Go port of model generation script...")

	// TODO: Fetch models from models.dev
	err := fetchModelsDev()
	if err != nil {
		log.Printf("Error fetching models.dev: %v\n", err)
	}

	// TODO: Fetch models from OpenRouter
	err = fetchOpenRouter()
	if err != nil {
		log.Printf("Error fetching OpenRouter: %v\n", err)
	}

	// TODO: Fetch models from Vercel AI Gateway
	err = fetchVercelGateway()
	if err != nil {
		log.Printf("Error fetching Vercel Gateway: %v\n", err)
	}

	// TODO: Write out generated Go structures or JSON file
	log.Println("Models generated successfully (stub).")
}

func fetchModelsDev() error {
	log.Println("Fetching models from models.dev API...")
	// Implementation stub:
	// resp, err := http.Get("https://models.dev/api/v1/models")
	// json.NewDecoder(resp.Body).Decode(&modelsDevResponse)
	return nil
}

func fetchOpenRouter() error {
	log.Println("Fetching models from OpenRouter API...")
	// Implementation stub:
	// resp, err := http.Get("https://openrouter.ai/api/v1/models")
	return nil
}

func fetchVercelGateway() error {
	log.Println("Fetching models from Vercel AI Gateway API...")
	// Implementation stub:
	// resp, err := http.Get("https://api.vercel.com/v1/ai/models")
	return nil
}
