package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerHealthCheck(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/route", nil)
	w := httptest.NewRecorder()

	Handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if data["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", data["status"])
	}
}

func TestHandlerOptionsCORS(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/api/route", nil)
	w := httptest.NewRecorder()

	Handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 for OPTIONS, got %d", resp.StatusCode)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected CORS header '*'")
	}
}

func TestHandlerInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/route", bytes.NewBufferString("not-a-json"))
	w := httptest.NewRecorder()

	Handler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for invalid JSON, got %d", resp.StatusCode)
	}
}
