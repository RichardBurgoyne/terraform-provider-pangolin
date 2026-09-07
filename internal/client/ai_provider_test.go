package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateAIProvider(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/org/acme/ai-provider" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&received)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"provider": map[string]any{
					"providerId":   float64(1),
					"orgId":        "acme",
					"name":         "OpenAI",
					"niceId":       "openai-1",
					"type":         "openai",
					"authType":     "bearer",
					"routingMode":  "url",
					"capabilities": []string{"openai_chat"},
					"apiKey":       "sk-secret",
					"enabled":      true,
				},
			},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	apiKey := "sk-secret"
	provider, err := c.CreateAIProvider(context.Background(), "acme", CreateAIProviderRequest{
		Name: "OpenAI", Type: "openai", APIKey: &apiKey, Capabilities: []string{"openai_chat"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.ProviderID != 1 || provider.APIKey != "sk-secret" {
		t.Errorf("unexpected provider: %+v", provider)
	}
	if received["type"] != "openai" {
		t.Errorf("expected type openai in request, got %#v", received)
	}
}

func TestUpdateAIProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/ai-provider/1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"provider": map[string]any{"providerId": float64(1), "name": "OpenAI Renamed", "enabled": false},
			},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	name := "OpenAI Renamed"
	enabled := false
	provider, err := c.UpdateAIProvider(context.Background(), 1, UpdateAIProviderRequest{Name: &name, Enabled: &enabled})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.Name != "OpenAI Renamed" || provider.Enabled {
		t.Errorf("unexpected provider: %+v", provider)
	}
}

func TestGetAIProvider_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "message": "not found"})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if _, err := c.GetAIProvider(context.Background(), 99); !IsNotFound(err) {
		t.Errorf("expected IsNotFound, got %v", err)
	}
}

func TestDeleteAIProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/ai-provider/1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteAIProvider(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
