// internal/client/client_test.go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDo_SetsAuthHeaderAndUnwrapsEnvelope(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"name": "hello"},
			"success": true,
			"error":   false,
			"message": "ok",
			"status":  200,
		})
	}))
	defer server.Close()

	c := New(server.URL, "test-token")

	var out struct {
		Name string `json:"name"`
	}
	if err := c.do(context.Background(), http.MethodGet, "/whatever", nil, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotAuth != "Bearer test-token" {
		t.Errorf("expected Authorization header %q, got %q", "Bearer test-token", gotAuth)
	}
	if out.Name != "hello" {
		t.Errorf("expected Name %q, got %q", "hello", out.Name)
	}
}

func TestDo_ReturnsAPIErrorOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":    nil,
			"success": false,
			"error":   true,
			"message": "Organization with ID x not found",
			"status":  404,
		})
	}))
	defer server.Close()

	c := New(server.URL, "test-token")

	err := c.do(context.Background(), http.MethodGet, "/org/x", nil, nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound(err) to be true, got false for error: %v", err)
	}
}
