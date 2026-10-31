package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func int64Pointer(v int64) *int64 { return &v }

func TestGetLabelPagesThroughList(t *testing.T) {
	var pages []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		pages = append(pages, page)
		labels := []any{}
		if page == "1" {
			for i := range listPageSize {
				labels = append(labels, map[string]any{"labelId": i + 100, "name": fmt.Sprint(i), "color": "#fff"})
			}
		} else {
			labels = append(labels, map[string]any{"labelId": 3, "name": "prod", "color": "#000"})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"labels": labels}, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	got, err := c.GetLabel(context.Background(), "acme", 3)
	if err != nil || got.Name != "prod" {
		t.Fatalf("get: %+v, %v", got, err)
	}
	if len(pages) != 2 {
		t.Errorf("expected 2 pages, fetched %v", pages)
	}
	if _, err := c.GetLabel(context.Background(), "acme", 99); !IsNotFound(err) {
		t.Errorf("expected not found for missing label, got %v", err)
	}
}

func TestLabelAttachment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{}
		labeled := []any{map[string]any{"labelId": 3}}
		switch r.URL.Path {
		case "/org/acme/label/3/attach", "/org/acme/label/3/detach":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["siteResourceId"] != float64(8) || len(body) != 1 {
				t.Errorf("unexpected body: %v", body)
			}
		case "/org/acme/sites":
			data["sites"] = []any{map[string]any{"siteId": 1, "labels": labeled}, map[string]any{"siteId": 2, "labels": []any{}}}
		case "/org/acme/resources":
			data["resources"] = []any{map[string]any{"resourceId": 5, "labels": labeled}}
		case "/org/acme/site-resources":
			data["siteResources"] = []any{map[string]any{"siteResourceId": 8, "labels": labeled}}
		case "/org/acme/clients":
			if r.URL.Query().Get("status") != "active,blocked,archived" {
				t.Errorf("expected every client status, got %q", r.URL.Query().Get("status"))
			}
			data["clients"] = []any{map[string]any{"clientId": 9, "labels": labeled}}
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	ctx := context.Background()
	target := LabelTarget{SiteResourceID: int64Pointer(8)}
	if err := c.AttachLabel(ctx, "acme", 3, target); err != nil {
		t.Fatalf("attach: %v", err)
	}
	if err := c.DetachLabel(ctx, "acme", 3, target); err != nil {
		t.Fatalf("detach: %v", err)
	}

	for name, tc := range map[string]struct {
		target LabelTarget
		want   bool
	}{
		"site attached":     {LabelTarget{SiteID: int64Pointer(1)}, true},
		"site not attached": {LabelTarget{SiteID: int64Pointer(2)}, false},
		"site missing":      {LabelTarget{SiteID: int64Pointer(3)}, false},
		"resource":          {LabelTarget{ResourceID: int64Pointer(5)}, true},
		"private resource":  {LabelTarget{SiteResourceID: int64Pointer(8)}, true},
		"client":            {LabelTarget{ClientID: int64Pointer(9)}, true},
	} {
		got, err := c.IsLabelAttached(ctx, "acme", 3, tc.target)
		if err != nil || got != tc.want {
			t.Errorf("%s: got %v, %v; want %v", name, got, err, tc.want)
		}
	}

	if _, err := c.IsLabelAttached(ctx, "acme", 3, LabelTarget{}); err == nil {
		t.Error("expected an error for a target with no ID")
	}
}
