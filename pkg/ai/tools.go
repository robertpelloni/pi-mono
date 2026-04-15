package ai

// OpenAIToolFunction represents the function payload for an OpenAI tool.
type OpenAIToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// Parameters is an arbitrary JSON Schema structure representing the function arguments.
	Parameters any  `json:"parameters"`
	Strict     bool `json:"strict,omitempty"`
}

// OpenAITool represents a tool definition as expected by the OpenAI Responses and Completions APIs.
type OpenAITool struct {
	Type     string             `json:"type"` // e.g. "function"
	Function OpenAIToolFunction `json:"function"`
}

// ConvertResponsesToolsOptions contains configuration options for converting tools.
type ConvertResponsesToolsOptions struct {
	Strict *bool
}

// ConvertResponsesTools translates the generic pi-ai Tool structs into the specific schema
// required by the OpenAI APIs (e.g. adding the "function" wrapper type).
func ConvertResponsesTools(tools []Tool, options *ConvertResponsesToolsOptions) []OpenAITool {
	var openAITools []OpenAITool

	strict := false
	if options != nil && options.Strict != nil {
		strict = *options.Strict
	}

	for _, tool := range tools {
		openAITools = append(openAITools, OpenAITool{
			Type: "function",
			Function: OpenAIToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
				Strict:      strict,
			},
		})
	}

	return openAITools
}
