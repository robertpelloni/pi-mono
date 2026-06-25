package ai

import (
	"context"
	"fmt"
)

// TabbyCompletionRequest represents the Tabby-compatible completion request schema.
type TabbyCompletionRequest struct {
	Language     string                 `json:"language"`
	Segments     *TabbySegments         `json:"segments,omitempty"`
	User         string                 `json:"user,omitempty"`
	DebugOptions *TabbyDebugOptions     `json:"debug_options,omitempty"`
	Temperature  *float32               `json:"temperature,omitempty"`
	Seed         *uint64                `json:"seed,omitempty"`
	Mode         string                 `json:"mode,omitempty"` // "standard" or "next_edit_suggestion"
}

type TabbySegments struct {
	Prefix                                     string             `json:"prefix"`
	Suffix                                     string             `json:"suffix,omitempty"`
	Filepath                                   string             `json:"filepath,omitempty"`
	GitURL                                     string             `json:"git_url,omitempty"`
	Declarations                               []TabbyDeclaration `json:"declarations,omitempty"`
	RelevantSnippetsFromChangedFiles           []TabbySnippet     `json:"relevant_snippets_from_changed_files,omitempty"`
	RelevantSnippetsFromRecentlyOpenedFiles    []TabbySnippet     `json:"relevant_snippets_from_recently_opened_files,omitempty"`
	Clipboard                                  string             `json:"clipboard,omitempty"`
	EditHistory                                *TabbyEditHistory  `json:"edit_history,omitempty"`
}

type TabbyDeclaration struct {
	Filepath string `json:"filepath"`
	Body     string `json:"body"`
}

type TabbySnippet struct {
	Filepath string  `json:"filepath"`
	Body     string  `json:"body"`
	Score    float32 `json:"score"`
}

type TabbyEditHistory struct {
	OriginalCode   string `json:"original_code"`
	EditsDiff      string `json:"edits_diff"`
	CurrentVersion string `json:"current_version"`
}

type TabbyDebugOptions struct {
	RawPrompt                               string `json:"raw_prompt,omitempty"`
	ReturnSnippets                          bool   `json:"return_snippets,omitempty"`
	ReturnPrompt                            bool   `json:"return_prompt,omitempty"`
	DisableRetrievalAugmentedCodeCompletion bool   `json:"disable_retrieval_augmented_code_completion,omitempty"`
}

type TabbyCompletionResponse struct {
	ID      string        `json:"id"`
	Choices []TabbyChoice `json:"choices"`
	Debug   *TabbyDebug   `json:"debug_data,omitempty"`
	Mode    string        `json:"mode"`
}

type TabbyChoice struct {
	Index int    `json:"index"`
	Text  string `json:"text"`
}

type TabbyDebug struct {
	Snippets []TabbySnippet `json:"snippets,omitempty"`
	Prompt   string         `json:"prompt,omitempty"`
}

// TabbyNextEditRequest represents the request for next edit suggestion.
type TabbyNextEditRequest struct {
	Segments *TabbySegments `json:"segments"`
	Filepath string         `json:"filepath"`
	Language string         `json:"language,omitempty"`
}

// TabbyNextEditResponse represents the response for next edit suggestion.
type TabbyNextEditResponse struct {
	Choice TabbyChoice `json:"choice"`
}

// HandleTabbyCompletion handles requests from Tabby-compatible clients.
func (r *Registry) HandleTabbyCompletion(ctx context.Context, req *TabbyCompletionRequest) (*TabbyCompletionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	// 1. Resolve prompt from segments
	prompt := ""
	if req.DebugOptions != nil && req.DebugOptions.RawPrompt != "" {
		prompt = req.DebugOptions.RawPrompt
	} else if req.Segments != nil {
		// FIM (Fill-In-the-Middle) prompt construction
		// Tabby uses specific markers like <pre>, <mid>, <end> depending on the model template.
		// For parity, we use a generic FIM structure if not specified.
		prompt = fmt.Sprintf("<PRE> %s <SUF> %s <MID>", req.Segments.Prefix, req.Segments.Suffix)
	} else if req.Mode == "next_edit_suggestion" && req.Segments != nil && req.Segments.EditHistory != nil {
		history := req.Segments.EditHistory
		prompt = fmt.Sprintf("Original:\n%s\n\nDiff:\n%s\n\nCurrent:\n%s\n\nNext edit:",
			history.OriginalCode, history.EditsDiff, history.CurrentVersion)
	}

	if prompt == "" {
		return nil, fmt.Errorf("empty prompt")
	}

	// 2. Map to unified Provider API
	// For now, we delegate to the default model in the registry.
	model := r.GetDefaultModel()
	if model == nil {
		return nil, fmt.Errorf("no default model configured")
	}

	fmt.Printf("Tabby using model: %s\n", model.ID)

	var temperature *float64
	if req.Temperature != nil {
		t := float64(*req.Temperature)
		temperature = &t
	}

	resp, err := model.Stream(ctx, Context{
		Messages: []Message{
			UserMessage{
				Content: []Content{
					TextContent{Text: prompt},
				},
			},
		},
	}, StreamOptions{
		Temperature: temperature,
	})
	if err != nil {
		return nil, err
	}

	// 3. Collect stream result (simplified for parity tool)
	fullText := ""
	for event := range resp {
		if event.Type == EventTextDelta && event.Delta != nil {
			fullText += *event.Delta
		}
	}

	mode := req.Mode
	if mode == "" {
		mode = "standard"
	}

	return &TabbyCompletionResponse{
		ID: fmt.Sprintf("tabby-%p", req),
		Choices: []TabbyChoice{
			{Index: 0, Text: fullText},
		},
		Mode: mode,
	}, nil
}

// HandleTabbyNextEdit handles next-edit suggestion requests.
func (r *Registry) HandleTabbyNextEdit(ctx context.Context, req *TabbyNextEditRequest) (*TabbyNextEditResponse, error) {
	if req == nil || req.Segments == nil {
		return nil, fmt.Errorf("invalid request")
	}

	// Delegate to completion handler with next_edit_suggestion mode
	compReq := &TabbyCompletionRequest{
		Language: req.Language,
		Segments: req.Segments,
		Mode:     "next_edit_suggestion",
	}

	resp, err := r.HandleTabbyCompletion(ctx, compReq)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choice returned from model")
	}

	return &TabbyNextEditResponse{
		Choice: resp.Choices[0],
	}, nil
}
