package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRedirectLifecycle(t *testing.T) {
	redirect := map[string]any{"redirectId": float64(7), "orgId": "acme", "niceId": "old-path", "name": "old", "resourceId": float64(5), "domainId": nil, "subdomain": nil, "destinationHost": "https://example.com", "pathMatchType": "prefix", "matchPath": "/old", "priority": float64(100), "ssl": true, "enabled": true}
	result := map[string]any{"data": map[string]any{"redirect": redirect}, "success": true}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/org/acme/redirect":
			_ = json.NewEncoder(w).Encode(result)
		case r.Method == http.MethodGet && r.URL.Path == "/org/acme/redirects/7":
			_ = json.NewEncoder(w).Encode(result)
		case r.Method == http.MethodPost && r.URL.Path == "/org/acme/redirects/7":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			// A cleared field must reach the server as an explicit null;
			// omitting it would leave the old value in place.
			if value, sent := body["matchPath"]; !sent || value != nil {
				t.Errorf("expected explicit null matchPath, got %v (sent: %v)", value, sent)
			}
			_ = json.NewEncoder(w).Encode(result)
		case r.Method == http.MethodDelete && r.URL.Path == "/org/acme/redirects/7":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(server.URL, "token")
	ctx := context.Background()
	created, err := c.CreateRedirect(ctx, "acme", RedirectRequest{Name: "old", DestinationHost: "https://example.com"})
	if err != nil || created.RedirectID != 7 {
		t.Fatalf("create: %+v, %v", created, err)
	}
	got, err := c.GetRedirect(ctx, "acme", 7)
	if err != nil || got.ResourceID == nil || *got.ResourceID != 5 || got.DomainID != nil {
		t.Fatalf("get: %+v, %v", got, err)
	}
	if _, err := c.UpdateRedirect(ctx, "acme", 7, RedirectRequest{Name: "old", DestinationHost: "https://example.com"}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := c.DeleteRedirect(ctx, "acme", 7); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
