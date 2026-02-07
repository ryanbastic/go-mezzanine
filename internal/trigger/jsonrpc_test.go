package trigger

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRPCClient_Call_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req.JSONRPC != "2.0" {
			t.Errorf("jsonrpc: got %q, want 2.0", req.JSONRPC)
		}
		if req.Method != "cell.written" {
			t.Errorf("method: got %q", req.Method)
		}

		resp := JSONRPCResponse{
			JSONRPC: "2.0",
			Result:  json.RawMessage(`"ok"`),
			ID:      req.ID,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewRPCClient(0, time.Millisecond, 5*time.Second)
	params := CellWrittenParams{
		AddedID:    42,
		RowKey:     "550e8400-e29b-41d4-a716-446655440000",
		ColumnName: "profile",
		RefKey:     1,
		Body:       json.RawMessage(`{"name":"test"}`),
		CreatedAt:  time.Now(),
		ShardID:    7,
	}

	resp, err := client.Call(context.Background(), srv.URL, "cell.written", params)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if resp.Error != nil {
		t.Errorf("unexpected RPC error: %v", resp.Error)
	}
}

func TestRPCClient_Call_RPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)

		resp := JSONRPCResponse{
			JSONRPC: "2.0",
			Error:   &JSONRPCError{Code: -32600, Message: "invalid request"},
			ID:      req.ID,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewRPCClient(0, time.Millisecond, 5*time.Second)
	resp, err := client.Call(context.Background(), srv.URL, "cell.written", nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected RPC error")
	}
	if resp.Error.Code != -32600 {
		t.Errorf("error code: got %d", resp.Error.Code)
	}
}

func TestRPCClient_Call_RetriesOn5xx(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		var req JSONRPCRequest
		json.NewDecoder(r.Body).Decode(&req)
		resp := JSONRPCResponse{JSONRPC: "2.0", Result: json.RawMessage(`"ok"`), ID: req.ID}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewRPCClient(3, time.Millisecond, 5*time.Second)
	resp, err := client.Call(context.Background(), srv.URL, "cell.written", nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}
	if attempts.Load() != 3 {
		t.Errorf("attempts: got %d, want 3", attempts.Load())
	}
}

func TestRPCClient_Call_MaxRetriesExhausted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewRPCClient(2, time.Millisecond, 5*time.Second)
	_, err := client.Call(context.Background(), srv.URL, "cell.written", nil)
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
}

func TestRPCClient_Call_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := NewRPCClient(0, time.Millisecond, 5*time.Second)

	_, err := client.Call(ctx, "http://localhost:1/rpc", "cell.written", nil)
	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
}

func TestJSONRPCError_Error(t *testing.T) {
	e := &JSONRPCError{Code: -32600, Message: "invalid request"}
	got := e.Error()
	want := "jsonrpc error -32600: invalid request"
	if got != want {
		t.Errorf("Error(): got %q, want %q", got, want)
	}
}
