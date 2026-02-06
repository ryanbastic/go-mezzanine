package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"key": "value"}
	writeJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", ct, "application/json")
	}

	var got map[string]string
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["key"] != "value" {
		t.Errorf("body: got %v", got)
	}
}

func TestWriteJSON_DifferentStatusCodes(t *testing.T) {
	codes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusBadRequest,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}

	for _, code := range codes {
		w := httptest.NewRecorder()
		writeJSON(w, code, map[string]string{})
		if w.Code != code {
			t.Errorf("expected status %d, got %d", code, w.Code)
		}
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, "invalid input")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp errorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error != "invalid input" {
		t.Errorf("error message: got %q, want %q", resp.Error, "invalid input")
	}
}

func TestWriteError_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusNotFound, "not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestWriteError_InternalServerError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusInternalServerError, "something broke")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d", w.Code)
	}

	var resp errorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error != "something broke" {
		t.Errorf("error: got %q", resp.Error)
	}
}

func TestWriteJSON_WithStruct(t *testing.T) {
	type response struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}

	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, response{Status: "ok", Count: 42})

	var got response
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Status != "ok" || got.Count != 42 {
		t.Errorf("got %+v", got)
	}
}

func TestWriteJSON_WithSlice(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, []int{1, 2, 3})

	var got []int
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("expected 3 elements, got %d", len(got))
	}
}

func TestWriteJSON_Nil(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, nil)

	if w.Code != http.StatusOK {
		t.Errorf("status: got %d", w.Code)
	}
}

func TestErrorResponse_JSON(t *testing.T) {
	resp := errorResponse{Error: "test error"}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["error"] != "test error" {
		t.Errorf("expected error field, got %v", got)
	}
}
