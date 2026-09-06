package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateDomain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/org/acme/domain" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"domainId": "d1"},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	out, err := c.CreateDomain(context.Background(), "acme", CreateDomainRequest{Type: "wildcard", BaseDomain: "example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.DomainID != "d1" {
		t.Errorf("expected domainId 'd1', got %q", out.DomainID)
	}
}

func TestGetDomain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/org/acme/domain/d1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"domainId": "d1", "baseDomain": "example.com", "type": "wildcard", "verified": true},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	d, err := c.GetDomain(context.Background(), "acme", "d1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !d.Verified {
		t.Error("expected domain to be verified")
	}
}

func TestDeleteDomain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/org/acme/domain/d1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteDomain(context.Background(), "acme", "d1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
