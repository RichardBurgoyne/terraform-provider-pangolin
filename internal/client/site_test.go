// internal/client/site_test.go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateSite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/org/acme/site" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"siteId": float64(1), "niceId": "site-1", "name": "Site 1", "type": "newt",
				"newtId": "newt-abc", "secret": "shh",
			},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	site, err := c.CreateSite(context.Background(), "acme", CreateSiteRequest{Name: "Site 1", Type: "newt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if site.SiteID != 1 || site.NewtID != "newt-abc" || site.Secret != "shh" {
		t.Errorf("unexpected site: %+v", site)
	}
}

func TestUpdateSite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/site/1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"siteId": float64(1), "name": "Renamed", "type": "newt"},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	name := "Renamed"
	site, err := c.UpdateSite(context.Background(), 1, UpdateSiteRequest{Name: &name})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if site.Name != "Renamed" {
		t.Errorf("expected name 'Renamed', got %q", site.Name)
	}
}

func TestDeleteSite(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/site/1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteSite(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
