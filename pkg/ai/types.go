package ai

// Api represents the type of API interface (e.g., "openai-completions", "anthropic-messages").
type Api string

const (
	ApiOpenAICompletions     Api = "openai-completions"
	ApiMistralConversations  Api = "mistral-conversations"
	ApiOpenAIResponses       Api = "openai-responses"
	ApiAzureOpenAIResponses  Api = "azure-openai-responses"
	ApiOpenAICodexResponses  Api = "openai-codex-responses"
	ApiAnthropicMessages     Api = "anthropic-messages"
	ApiBedrockConverseStream Api = "bedrock-converse-stream"
	ApiGoogleGenerativeAI    Api = "google-generative-ai"
	ApiGoogleGeminiCLI       Api = "google-gemini-cli"
	ApiGoogleVertex          Api = "google-vertex"
)

// Provider represents the service provider offering the model.
type Provider string

const (
	ProviderAmazonBedrock Provider = "amazon-bedrock"
	ProviderAnthropic     Provider = "anthropic"
	ProviderGoogle        Provider = "google"
	ProviderOpenAI        Provider = "openai"
	ProviderAzureOpenAI   Provider = "azure-openai-responses"
	ProviderGroq          Provider = "groq"
	ProviderCerebras      Provider = "cerebras"
	ProviderXAI           Provider = "xai"
	ProviderZAI           Provider = "zai"
	ProviderMistral       Provider = "mistral"
	ProviderHuggingFace   Provider = "huggingface"
	ProviderOpenCode      Provider = "opencode"
	ProviderOpenCodeGo    Provider = "opencode-go"
	ProviderGithubCopilot Provider = "github-copilot"
	ProviderMinimax       Provider = "minimax"
	ProviderMinimaxCN     Provider = "minimax-cn"
	ProviderKimiCoding    Provider = "kimi-coding"
	ProviderOpenRouter    Provider = "openrouter"
	ProviderVercelGateway Provider = "vercel-ai-gateway"
	ProviderOpenAICodex   Provider = "openai-codex"
	ProviderGeminiCLI     Provider = "google-gemini-cli"
	ProviderAntigravity   Provider = "google-antigravity"
	ProviderVertex        Provider = "google-vertex"
)

// ThinkingLevel specifies the amount of reasoning effort for supported models.
type ThinkingLevel string

const (
	ThinkingLow    ThinkingLevel = "low"
	ThinkingMedium ThinkingLevel = "medium"
	ThinkingHigh   ThinkingLevel = "high"
)

type ThinkingBudgets struct {
	Low    int `json:"low"`
	Medium int `json:"medium"`
	High   int `json:"high"`
}

type CacheRetention string

const (
	CacheRetentionNone  CacheRetention = "none"
	CacheRetentionShort CacheRetention = "short"
	CacheRetentionLong  CacheRetention = "long"
)

// StreamOptions contains general options for streaming API requests.
type StreamOptions struct {
	AbortSignal     <-chan struct{}   `json:"-"`
	SystemPrompt    string            `json:"systemPrompt,omitempty"`
	Temperature     *float64          `json:"temperature,omitempty"`
	MaxTokens       int               `json:"maxTokens,omitempty"`
	CacheRetention  CacheRetention    `json:"cacheRetention,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	MaxRetryDelayMs int               `json:"maxRetryDelayMs,omitempty"`
	Metadata        map[string]any    `json:"metadata,omitempty"`
}

// SimpleStreamOptions unifies options with reasoning support.
type SimpleStreamOptions struct {
	StreamOptions
	Reasoning       ThinkingLevel    `json:"reasoning,omitempty"`
	ThinkingBudgets *ThinkingBudgets `json:"thinkingBudgets,omitempty"`
}

type ContentType string

const (
	ContentTypeText     ContentType = "text"
	ContentTypeThinking ContentType = "thinking"
	ContentTypeImage    ContentType = "image"
	ContentTypeToolCall ContentType = "toolCall"
)

type Content interface {
	GetType() ContentType
}

type TextContent struct {
	Text          string  `json:"text"`
	TextSignature *string `json:"textSignature,omitempty"`
}

func (c TextContent) GetType() ContentType { return ContentTypeText }

type ThinkingContent struct {
	Thinking          string  `json:"thinking"`
	ThinkingSignature *string `json:"thinkingSignature,omitempty"`
	Redacted          *bool   `json:"redacted,omitempty"`
}

func (c ThinkingContent) GetType() ContentType { return ContentTypeThinking }

type ImageContent struct {
	Data     string `json:"data"`     // Base64 encoded image data
	MimeType string `json:"mimeType"` // e.g., "image/jpeg", "image/png"
}

func (c ImageContent) GetType() ContentType { return ContentTypeImage }

type ToolCall struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Arguments        map[string]any `json:"arguments"`
	ThoughtSignature *string        `json:"thoughtSignature,omitempty"`
}

func (c ToolCall) GetType() ContentType { return ContentTypeToolCall }

type Usage struct {
	Input       int       `json:"input"`
	Output      int       `json:"output"`
	CacheRead   int       `json:"cacheRead"`
	CacheWrite  int       `json:"cacheWrite"`
	TotalTokens int       `json:"totalTokens"`
	Cost        UsageCost `json:"cost"`
}

type UsageCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
	Total      float64 `json:"total"`
}

type StopReason string

const (
	StopReasonStop    StopReason = "stop"
	StopReasonLength  StopReason = "length"
	StopReasonToolUse StopReason = "toolUse"
	StopReasonError   StopReason = "error"
	StopReasonAborted StopReason = "aborted"
)

type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "toolResult"
)

type Message interface {
	GetRole() MessageRole
}

type UserMessage struct {
	Content   []Content `json:"content"` // TextContent | ImageContent
	Timestamp int64     `json:"timestamp"`
}

func (m UserMessage) GetRole() MessageRole { return RoleUser }

type AssistantMessage struct {
	Content      []Content  `json:"content"` // TextContent | ThinkingContent | ToolCall
	API          Api        `json:"api"`
	Provider     Provider   `json:"provider"`
	Model        string     `json:"model"`
	ResponseID   *string    `json:"responseId,omitempty"`
	Usage        Usage      `json:"usage"`
	StopReason   StopReason `json:"stopReason"`
	ErrorMessage *string    `json:"errorMessage,omitempty"`
	Timestamp    int64      `json:"timestamp"`
}

func (m AssistantMessage) GetRole() MessageRole { return RoleAssistant }

type ToolResultMessage struct {
	ToolCallID string    `json:"toolCallId"`
	ToolName   string    `json:"toolName"`
	Content    []Content `json:"content"` // TextContent | ImageContent
	Details    any       `json:"details,omitempty"`
	IsError    bool      `json:"isError"`
	Timestamp  int64     `json:"timestamp"`
}

func (m ToolResultMessage) GetRole() MessageRole { return RoleTool }

type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"` // Typically represents a JSON Schema map
}

type Context struct {
	SystemPrompt *string   `json:"systemPrompt,omitempty"`
	Messages     []Message `json:"messages"`
	Tools        []Tool    `json:"tools,omitempty"`
}

type ModelCost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

type OpenAICompletionsCompat struct {
	SupportsStore                    *bool             `json:"supportsStore,omitempty"`
	SupportsDeveloperRole            *bool             `json:"supportsDeveloperRole,omitempty"`
	SupportsReasoningEffort          *bool             `json:"supportsReasoningEffort,omitempty"`
	ReasoningEffortMap               map[string]string `json:"reasoningEffortMap,omitempty"`
	SupportsUsageInStreaming         *bool             `json:"supportsUsageInStreaming,omitempty"`
	MaxTokensField                   *string           `json:"maxTokensField,omitempty"`
	RequiresToolResultName           *bool             `json:"requiresToolResultName,omitempty"`
	RequiresAssistantAfterToolResult *bool             `json:"requiresAssistantAfterToolResult,omitempty"`
	RequiresThinkingAsText           *bool             `json:"requiresThinkingAsText,omitempty"`
	ThinkingFormat                   *string           `json:"thinkingFormat,omitempty"`
	OpenRouterRouting                *any              `json:"openRouterRouting,omitempty"`
	VercelGatewayRouting             *any              `json:"vercelGatewayRouting,omitempty"`
	ZaiToolStream                    *bool             `json:"zaiToolStream,omitempty"`
	SupportsStrictMode               *bool             `json:"supportsStrictMode,omitempty"`
}

type ModelInfo struct {
	ID            string                   `json:"id"`
	Name          string                   `json:"name"`
	API           Api                      `json:"api"`
	Provider      Provider                 `json:"provider"`
	BaseURL       string                   `json:"baseUrl"`
	Reasoning     bool                     `json:"reasoning"`
	Input         []string                 `json:"input"` // "text", "image"
	Cost          ModelCost                `json:"cost"`
	ContextWindow int                      `json:"contextWindow"`
	MaxTokens     int                      `json:"maxTokens"`
	Headers       map[string]string        `json:"headers,omitempty"`
	Compat        *OpenAICompletionsCompat `json:"compat,omitempty"`
}
