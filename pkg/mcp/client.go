package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sync"
)

type TransportType string

const (
	TransportStdio TransportType = "stdio"
	TransportSSE   TransportType = "sse"
)

type Client struct {
	transport TransportType
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    *bufio.Scanner
	sseURL    string
	httpClient *http.Client
	mu        sync.Mutex
	pending   map[interface{}]chan *Response
	nextID    int
}

func NewStdioClient(command string, args ...string) (*Client, error) {
	cmd := exec.Command(command, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil { return nil, err }
	stdout, err := cmd.StdoutPipe()
	if err != nil { return nil, err }
	if err := cmd.Start(); err != nil { return nil, err }

	c := &Client{
		transport: TransportStdio,
		cmd:       cmd,
		stdin:     stdin,
		stdout:    bufio.NewScanner(stdout),
		pending:   make(map[interface{}]chan *Response),
		nextID:    1,
	}
	go c.readLoop()
	return c, nil
}

func NewSSEClient(url string) (*Client, error) {
	c := &Client{
		transport: TransportSSE,
		sseURL:    url,
		httpClient: &http.Client{},
		pending:   make(map[interface{}]chan *Response),
		nextID:    1,
	}
	// In a real implementation we would start an SSE listener here
	return c, nil
}

func (c *Client) readLoop() {
	if c.transport != TransportStdio { return }
	for c.stdout.Scan() {
		var resp Response
		if err := json.Unmarshal(c.stdout.Bytes(), &resp); err != nil { continue }
		c.mu.Lock()
		if ch, ok := c.pending[resp.ID]; ok {
			ch <- &resp
			delete(c.pending, resp.ID)
		}
		c.mu.Unlock()
	}
}

func (c *Client) Call(ctx context.Context, method string, params interface{}) (*Response, error) {
	c.mu.Lock()
	id := c.nextID
	c.nextID++
	ch := make(chan *Response, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	paramsJSON, _ := json.Marshal(params)
	req := Request{JSONRPC: "2.0", ID: id, Method: method, Params: paramsJSON}
	reqBytes, _ := json.Marshal(req)

	if c.transport == TransportStdio {
		fmt.Fprintf(c.stdin, "%s\n", string(reqBytes))
	} else {
		resp, err := c.httpClient.Post(c.sseURL, "application/json", bytes.NewBuffer(reqBytes))
		if err != nil { return nil, err }
		defer resp.Body.Close()
		var mcpResp Response
		json.NewDecoder(resp.Body).Decode(&mcpResp)
		return &mcpResp, nil
	}

	select {
	case <-ctx.Done(): return nil, ctx.Err()
	case resp := <-ch: return resp, nil
	}
}

func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	resp, err := c.Call(ctx, "tools/list", nil)
	if err != nil { return nil, err }
	var result ListToolsResult
	json.Unmarshal(resp.Result, &result)
	return result.Tools, nil
}

func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*CallToolResult, error) {
	resp, err := c.Call(ctx, "tools/call", CallToolRequest{Name: name, Arguments: args})
	if err != nil { return nil, err }
	var result CallToolResult
	json.Unmarshal(resp.Result, &result)
	return &result, nil
}

func (c *Client) Close() error {
	if c.transport == TransportStdio {
		c.stdin.Close()
		return c.cmd.Process.Kill()
	}
	return nil
}
