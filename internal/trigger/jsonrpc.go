package trigger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync/atomic"
	"time"
)

// JSONRPCRequest is a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
	ID      int64  `json:"id"`
}

// JSONRPCResponse is a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
	ID      int64           `json:"id"`
}

// JSONRPCError represents a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *JSONRPCError) Error() string {
	return fmt.Sprintf("jsonrpc error %d: %s", e.Code, e.Message)
}

// CellWrittenParams is the notification payload sent to plugins.
type CellWrittenParams struct {
	AddedID    int64           `json:"added_id"`
	RowKey     string          `json:"row_key"`
	ColumnName string          `json:"column_name"`
	RefKey     int64           `json:"ref_key"`
	Body       json.RawMessage `json:"body"`
	CreatedAt  time.Time       `json:"created_at"`
	ShardID    int             `json:"shard_id"`
}

// RPCClient sends JSON-RPC 2.0 requests over HTTP with retries.
type RPCClient struct {
	httpClient *http.Client
	nextID     atomic.Int64
	maxRetries int
	baseDelay  time.Duration
}

// NewRPCClient creates a client with the given retry settings and timeout.
func NewRPCClient(maxRetries int, baseDelay time.Duration, timeout time.Duration) *RPCClient {
	return &RPCClient{
		httpClient: &http.Client{Timeout: timeout},
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
	}
}

// Call sends a JSON-RPC 2.0 request to endpoint. Retries on 5xx/network errors.
func (c *RPCClient) Call(ctx context.Context, endpoint, method string, params any) (*JSONRPCResponse, error) {
	id := c.nextID.Add(1)
	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      id,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal rpc request: %w", err)
	}

	var lastErr error
	for attempt := range c.maxRetries + 1 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		resp, err := c.doRequest(ctx, endpoint, data)
		if err == nil {
			return resp, nil
		}
		lastErr = err

		if attempt < c.maxRetries {
			delay := c.baseDelay * time.Duration(math.Pow(2, float64(attempt)))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return nil, fmt.Errorf("rpc call failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

func (c *RPCClient) doRequest(ctx context.Context, endpoint string, data []byte) (*JSONRPCResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("server error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var rpcResp JSONRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("unmarshal rpc response: %w", err)
	}

	return &rpcResp, nil
}
