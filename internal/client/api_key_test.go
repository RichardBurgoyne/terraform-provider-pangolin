package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/org/acme/api-key" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"apiKeyId": "k1", "name": "ci", "apiKey": "secret-value", "lastChars": "alue", "createdAt": "2026-01-01T00:00:00Z"},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	key, err := c.CreateAPIKey(context.Background(), "acme", CreateAPIKeyRequest{Name: "ci"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if key.APIKey != "secret-value" {
		t.Errorf("expected apiKey 'secret-value', got %q", key.APIKey)
	}
}

func TestSetAPIKeyActions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/org/acme/api-key/k1/actions" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.SetAPIKeyActions(context.Background(), "acme", "k1", []string{"listSites"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/org/acme/api-key/k1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteAPIKey(context.Background(), "acme", "k1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
