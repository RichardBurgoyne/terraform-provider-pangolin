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
		json.NewEncoder(w).Encode(map[string]any{
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

func TestDeleteResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/resource/5" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteResource(context.Background(), 5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
