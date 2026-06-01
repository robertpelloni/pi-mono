package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// Client handles communication with an MCP server via stdio.
type Client struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Scanner
	mu      sync.Mutex
	pending map[interface{}]chan *Response
	nextID  int
}

// NewClient starts a new MCP server process and initializes the client.
func NewClient(command string, args ...string) (*Client, error) {
	cmd := exec.Command(command, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	c := &Client{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewScanner(stdout),
		pending: make(map[interface{}]chan *Response),
		nextID:  1,
	}

	go c.readLoop()
	return c, nil
}

func (c *Client) readLoop() {
	for c.stdout.Scan() {
		var resp Response
		if err := json.Unmarshal(c.stdout.Bytes(), &resp); err != nil {
			continue
		}

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
	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  paramsJSON,
	}

	reqBytes, _ := json.Marshal(req)
	fmt.Fprintf(c.stdin, "%s\n", string(reqBytes))

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-ch:
		return resp, nil
	}
}

func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	resp, err := c.Call(ctx, "tools/list", nil)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("MCP error: %s", resp.Error.Message)
	}

	var result ListToolsResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}
	return result.Tools, nil
}

func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*CallToolResult, error) {
	req := CallToolRequest{
		Name:      name,
		Arguments: args,
	}
	resp, err := c.Call(ctx, "tools/call", req)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("MCP error: %s", resp.Error.Message)
	}

	var result CallToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) Close() error {
	c.stdin.Close()
	return c.cmd.Process.Kill()
}
