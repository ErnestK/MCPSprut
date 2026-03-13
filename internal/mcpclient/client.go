package mcpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/ErnestK/mcp-sprut/internal/jsonrpc"
	"github.com/ErnestK/mcp-sprut/internal/storage"
)

const (
	protocolVersion = "2024-11-05"
	clientName      = "mcp-sprut"
	clientVersion   = "0.1.0"
)

type Client struct {
	httpClient *http.Client
	idCounter  atomic.Int64
}

func NewClient(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *Client) nextID() int {
	return int(c.idCounter.Add(1))
}

func (c *Client) post(ctx context.Context, url string, method string, params interface{}) (json.RawMessage, error) {
	req, err := jsonrpc.NewRequest(c.nextID(), method, params)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	var rpcResp jsonrpc.Response
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, nil
}

func (c *Client) Initialize(ctx context.Context, serverURL string) (*jsonrpc.InitializeResult, error) {
	raw, err := c.post(ctx, serverURL, "initialize", map[string]interface{}{
		"protocolVersion": protocolVersion,
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    clientName,
			"version": clientVersion,
		},
	})
	if err != nil {
		return nil, err
	}

	var result jsonrpc.InitializeResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse initialize result: %w", err)
	}

	return &result, nil
}

func (c *Client) SendInitialized(ctx context.Context, serverURL string) error {
	notification := jsonrpc.Notification{
		JSONRPC: jsonrpc.Version,
		Method:  "notifications/initialized",
	}

	body, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	resp.Body.Close()

	return nil
}

func (c *Client) ListTools(ctx context.Context, serverURL string) ([]storage.Tool, error) {
	raw, err := c.post(ctx, serverURL, "tools/list", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Tools []storage.Tool `json:"tools"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parse tools list: %w", err)
	}

	return result.Tools, nil
}
