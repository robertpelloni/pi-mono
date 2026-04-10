package tools

import (
	"context"
	"encoding/json"

	"github.com/mariozechner/pi-mono/go-packages/ai/types"
)

// LegacyWrapper wraps an existing AgentTool to provide exact schema parity under a different name.
type LegacyWrapper struct {
	Original    AgentTool
	NewName     string
	Description string
}

func (w *LegacyWrapper) Definition() types.Tool {
	origDef := w.Original.Definition()
	return types.Tool{
		Name:        w.NewName,
		Description: w.Description,
		Parameters:  origDef.Parameters,
	}
}

func (w *LegacyWrapper) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return w.Original.Execute(ctx, args)
}

// Ensure LegacyWrapper implements AgentTool
var _ AgentTool = (*LegacyWrapper)(nil)
