package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/resource/5/target" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"targetId": float64(9), "siteId": float64(1), "ip": "10.0.0.5", "port": float64(80), "enabled": true},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	target, err := c.CreateTarget(context.Background(), 5, CreateTargetRequest{SiteID: 1, IP: "10.0.0.5", Port: 80})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.TargetID != 9 {
		t.Errorf("expected targetId 9, got %d", target.TargetID)
	}
}

func TestDeleteTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/target/9" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteTarget(context.Background(), 9); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
