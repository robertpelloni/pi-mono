package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
)

// Position represents a position in a text document (0-indexed).
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range represents a range in a text document.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location represents a location in a text document.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// Diagnostic represents a diagnostic from the language server.
type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity"`
	Message  string `json:"message"`
	Source   string `json:"source,omitempty"`
	Code     any    `json:"code,omitempty"`
}

// DocumentSymbol represents a symbol in a document.
type DocumentSymbol struct {
	Name           string           `json:"name"`
	Kind           int              `json:"kind"`
	Range          Range            `json:"range"`
	SelectionRange Range            `json:"selectionRange"`
	Children       []DocumentSymbol `json:"children,omitempty"`
}

// CompletionItem represents a completion item.
type CompletionItem struct {
	Label  string `json:"label"`
	Kind   int    `json:"kind,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// RPC types
type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// ClientState represents the LSP client connection state.
type ClientState int

const (
	ClientStopped  ClientState = iota
	ClientStarting
	ClientRunning
	ClientError
)

// Client is an LSP client that communicates over stdin/stdout with a language server.
type Client struct {
	mu          sync.Mutex
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	stdout      io.ReadCloser
	nextID      atomic.Int64
	pending     map[int]chan *rpcResponse
	state       ClientState
	handlers    map[string]func(any)
	diagnostics map[string][]Diagnostic
}

// ClientConfig defines how to start a language server.
type ClientConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
	RootURI string            `json:"rootURI"`
}

// NewClient creates a new LSP client.
func NewClient() *Client {
	return &Client{
		pending:     make(map[int]chan *rpcResponse),
		handlers:    make(map[string]func(any)),
		diagnostics: make(map[string][]Diagnostic),
	}
}

// Start launches the language server process.
func (c *Client) Start(ctx context.Context, config ClientConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cmd = exec.CommandContext(ctx, config.Command, config.Args...)

	if len(config.Env) > 0 {
		env := c.cmd.Environ()
		for k, v := range config.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		c.cmd.Env = env
	}

	stdin, err := c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	c.stdin = stdin

	stdout, err := c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	c.stdout = stdout

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	c.state = ClientStarting
	go c.readLoop()

	// Initialize LSP session
	initParams := map[string]any{
		"processId":    0,
		"rootUri":      config.RootURI,
		"capabilities": defaultCapabilities(),
	}
	if _, err := c.call("initialize", initParams); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	c.notify("initialized", map[string]any{})
	c.state = ClientRunning
	return nil
}

// Stop shuts down the language server.
func (c *Client) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == ClientStopped {
		return nil
	}
	c.call("shutdown", nil) //nolint:errcheck
	c.notify("exit", nil)

	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.stdout != nil {
		c.stdout.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill() //nolint:errcheck
	}
	c.state = ClientStopped
	return nil
}

// State returns the current client state.
func (c *Client) State() ClientState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// DidOpen notifies the server that a file was opened.
func (c *Client) DidOpen(uri, languageID, text string) error {
	return c.notify("textDocument/didOpen", map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": languageID,
			"version":    0,
			"text":       text,
		},
	})
}

// DidChange notifies the server that a file was modified.
func (c *Client) DidChange(uri string, version int, text string) error {
	return c.notify("textDocument/didChange", map[string]any{
		"textDocument": map[string]any{
			"uri":     uri,
			"version": version,
		},
		"contentChanges": []map[string]any{
			{"text": text},
		},
	})
}

// Diagnostics returns the latest diagnostics for a URI.
func (c *Client) Diagnostics(uri string) []Diagnostic {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.diagnostics[uri]
}

// Hover requests hover information at a position.
func (c *Client) Hover(ctx context.Context, uri string, pos Position) (string, error) {
	result, err := c.call("textDocument/hover", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     pos,
	})
	if err != nil {
		return "", err
	}
	return extractHoverContent(result)
}

// GoToDefinition requests the definition location for a symbol.
func (c *Client) GoToDefinition(ctx context.Context, uri string, pos Position) ([]Location, error) {
	result, err := c.call("textDocument/definition", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     pos,
	})
	if err != nil {
		return nil, err
	}

	var locations []Location
	if err := json.Unmarshal(result, &locations); err != nil {
		var single Location
		if err2 := json.Unmarshal(result, &single); err2 != nil {
			return nil, err
		}
		return []Location{single}, nil
	}
	return locations, nil
}

// References requests all references to a symbol.
func (c *Client) References(ctx context.Context, uri string, pos Position) ([]Location, error) {
	result, err := c.call("textDocument/references", map[string]any{
		"textDocument": map[string]any{"uri": uri},
		"position":     pos,
		"context":      map[string]any{"includeDeclaration": true},
	})
	if err != nil {
		return nil, err
	}

	var locations []Location
	if err := json.Unmarshal(result, &locations); err != nil {
		return nil, err
	}
	return locations, nil
}

// DocumentSymbols requests all symbols in a document.
func (c *Client) DocumentSymbols(ctx context.Context, uri string) ([]DocumentSymbol, error) {
	result, err := c.call("textDocument/documentSymbol", map[string]any{
		"textDocument": map[string]any{"uri": uri},
	})
	if err != nil {
		return nil, err
	}

	var symbols []DocumentSymbol
	if err := json.Unmarshal(result, &symbols); err != nil {
		return nil, err
	}
	return symbols, nil
}

// OnNotification registers a handler for a notification method.
func (c *Client) OnNotification(method string, handler func(any)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[method] = handler
}

// ── Internal ──

func (c *Client) call(method string, params any) (json.RawMessage, error) {
	id := int(c.nextID.Add(1))
	ch := make(chan *rpcResponse, 1)

	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	req := rpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	if err := c.send(req); err != nil {
		return nil, fmt.Errorf("send: %w", err)
	}

	resp := <-ch
	if resp.Error != nil {
		return nil, fmt.Errorf("LSP error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	return resp.Result, nil
}

func (c *Client) notify(method string, params any) error {
	return c.send(rpcNotification{JSONRPC: "2.0", Method: method, Params: params})
}

func (c *Client) send(msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, err := io.WriteString(c.stdin, header); err != nil {
		return err
	}
	_, err = c.stdin.Write(data)
	return err
}

func (c *Client) readLoop() {
	buf := make([]byte, 4096)
	var contentBuf []byte
	var contentLength int

	for {
		// Read header
		headerBuf := make([]byte, 0)
		for {
			n, err := c.stdout.Read(buf)
			if err != nil {
				return
			}
			headerBuf = append(headerBuf, buf[:n]...)
			header := string(headerBuf)
			if _, err := fmt.Sscanf(header, "Content-Length: %d\r\n", &contentLength); err == nil {
				idx := findHeaderEnd(headerBuf)
				if idx >= 0 {
					contentBuf = headerBuf[idx:]
					break
				}
			}
		}

		// Read body
		for len(contentBuf) < contentLength {
			n, err := c.stdout.Read(buf)
			if err != nil {
				return
			}
			contentBuf = append(contentBuf, buf[:n]...)
		}

		if contentLength > 0 && len(contentBuf) >= contentLength {
			body := contentBuf[:contentLength]
			c.handleMessage(body)
			contentBuf = contentBuf[contentLength:]
		}
	}
}

func (c *Client) handleMessage(data []byte) {
	var resp rpcResponse
	if err := json.Unmarshal(data, &resp); err == nil && resp.ID > 0 {
		c.mu.Lock()
		ch, ok := c.pending[resp.ID]
		c.mu.Unlock()
		if ok {
			ch <- &resp
		}
		return
	}

	var notif rpcNotification
	if err := json.Unmarshal(data, &notif); err == nil && notif.Method != "" {
		c.mu.Lock()
		handler, ok := c.handlers[notif.Method]
		c.mu.Unlock()
		if ok && handler != nil {
			handler(notif.Params)
		}
		if notif.Method == "textDocument/publishDiagnostics" {
			c.handleDiagnostics(notif.Params)
		}
	}
}

func (c *Client) handleDiagnostics(params any) {
	data, err := json.Marshal(params)
	if err != nil {
		return
	}
	var diag struct {
		URI         string       `json:"uri"`
		Diagnostics []Diagnostic `json:"diagnostics"`
	}
	if err := json.Unmarshal(data, &diag); err != nil {
		return
	}
	c.mu.Lock()
	c.diagnostics[diag.URI] = diag.Diagnostics
	c.mu.Unlock()
}

func extractHoverContent(raw json.RawMessage) (string, error) {
	var hover struct {
		Contents any `json:"contents"`
	}
	if err := json.Unmarshal(raw, &hover); err != nil {
		return "", err
	}
	switch v := hover.Contents.(type) {
	case string:
		return v, nil
	case map[string]any:
		if value, ok := v["value"].(string); ok {
			return value, nil
		}
	}
	return fmt.Sprintf("%v", hover.Contents), nil
}

func findHeaderEnd(data []byte) int {
	for i := 0; i < len(data)-3; i++ {
		if data[i] == '\r' && data[i+1] == '\n' && data[i+2] == '\r' && data[i+3] == '\n' {
			return i + 4
		}
	}
	return -1
}

func defaultCapabilities() map[string]any {
	return map[string]any{
		"textDocument": map[string]any{
			"completion": map[string]any{
				"completionItem": map[string]any{"snippetSupport": false},
			},
			"hover": map[string]any{
				"contentFormat": []string{"plaintext", "markdown"},
			},
			"publishDiagnostics": map[string]any{
				"relatedInformation": true,
			},
		},
	}
}
