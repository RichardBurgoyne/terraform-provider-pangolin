package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/org/acme/resource" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"resourceId": float64(5), "niceId": "res-1", "name": "Res 1", "mode": "http",
				"domainId": "d1", "fullDomain": "res-1.example.com", "enabled": true, "ssl": true,
			},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	res, err := c.CreateResource(context.Background(), "acme", CreateResourceRequest{Name: "Res 1", Mode: "http", DomainID: "d1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ResourceID != 5 || res.FullDomain != "res-1.example.com" {
		t.Errorf("unexpected resource: %+v", res)
	}
}

func TestCreateResource_RawTCPUDP(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"resourceId": float64(6), "niceId": "res-2", "name": "Raw TCP", "mode": "tcp",
				"proxyPort": float64(5432),
			},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	proxyPort := int64(5432)
	res, err := c.CreateResource(context.Background(), "acme", CreateResourceRequest{Name: "Raw TCP", Mode: "tcp", ProxyPort: &proxyPort})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ProxyPort != 5432 || res.Mode != "tcp" {
		t.Errorf("unexpected resource: %+v", res)
	}
	if _, ok := received["domainId"]; ok {
		t.Errorf("expected domainId to be omitted from a raw resource create request, got %#v", received)
	}
}

func TestUpdateResource_RawTCPUDP_OmitsHttpOnlyFields(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"resourceId": float64(6), "mode": "tcp", "proxyPort": float64(5433)},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	proxyPort := int64(5433)
	name := "Raw TCP"
	if _, err := c.UpdateResource(context.Background(), 6, UpdateResourceRequest{Name: &name, ProxyPort: &proxyPort}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, httpOnly := range []string{"domainId", "ssl", "postAuthPath", "pamMode", "authDaemonMode", "authDaemonPort", "subdomain"} {
		if _, ok := received[httpOnly]; ok {
			t.Errorf("expected %q to be omitted from a raw resource update request, got %#v", httpOnly, received)
		}
	}
	if received["proxyPort"] != float64(5433) {
		t.Errorf("expected proxyPort 5433 in request, got %#v", received["proxyPort"])
	}
}

func TestCreateResource_InferenceModeWithAIProviders(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"resourceId": float64(7), "niceId": "res-3", "name": "AI Gateway", "mode": "inference",
				"domainId": "d1", "fullDomain": "ai.example.com",
			},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	res, err := c.CreateResource(context.Background(), "acme", CreateResourceRequest{
		Name: "AI Gateway", Mode: "inference", DomainID: "d1",
		AIProviders: []ResourceAIProviderAttachment{{ProviderID: 1, AccessMode: "inherit"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Mode != "inference" {
		t.Errorf("unexpected resource: %+v", res)
	}
	providers, ok := received["aiProviders"].([]any)
	if !ok || len(providers) != 1 {
		t.Errorf("expected one aiProviders entry in request, got %#v", received["aiProviders"])
	}
}

func TestSetAndListResourceAIProviders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/resource/7/ai-providers":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			providers, _ := body["providers"].([]any)
			if len(providers) != 1 {
				t.Errorf("expected one provider in set request, got %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}, "success": true})
		case r.Method == http.MethodGet && r.URL.Path == "/resource/7/ai-providers":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"providers": []map[string]any{
						{"providerId": float64(1), "name": "OpenAI", "type": "custom", "enabled": true, "providerEnabled": true, "accessMode": "inherit"},
					},
				},
				"success": true,
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.SetResourceAIProviders(context.Background(), 7, []ResourceAIProviderAttachment{{ProviderID: 1, AccessMode: "inherit"}}); err != nil {
		t.Fatalf("unexpected error setting providers: %v", err)
	}

	providers, err := c.ListResourceAIProviders(context.Background(), 7)
	if err != nil {
		t.Fatalf("unexpected error listing providers: %v", err)
	}
	if len(providers) != 1 || providers[0].ProviderID != 1 || providers[0].AccessMode != "inherit" {
		t.Errorf("unexpected providers: %+v", providers)
	}
}

func TestDeleteResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/resource/5" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteResource(context.Background(), 5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
