package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateSiteResource_SendsEmptyAssignments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/org/acme/site-resource" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		for _, k := range []string{"userIds", "roleIds", "clientIds"} {
			if arr, ok := body[k].([]any); !ok || len(arr) != 0 {
				t.Errorf("%s should be sent as an empty array, got %#v", k, body[k])
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"siteResourceId": float64(12), "name": "db", "mode": "host"}, "success": true})
	}))
	defer server.Close()

	sr, err := New(server.URL, "token").CreateSiteResource(context.Background(), "acme", CreateSiteResourceRequest{Name: "db", Mode: "host", SiteIDs: []int64{1}})
	if err != nil || sr.SiteResourceID != 12 {
		t.Fatalf("unexpected result: %+v, %v", sr, err)
	}
}

func TestListSiteResourceSiteIDs_Paginates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		items := []any{map[string]any{"siteResourceId": float64(1), "siteIds": []any{float64(9)}}}
		if page == "2" {
			items = []any{map[string]any{"siteResourceId": float64(12), "siteIds": []any{float64(3), float64(4)}}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"siteResources": items, "pagination": map[string]any{"total": float64(150)}}, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	ids, err := c.ListSiteResourceSiteIDs(context.Background(), "acme", 12)
	if err != nil || len(ids) != 2 || ids[0] != 3 {
		t.Fatalf("unexpected ids: %v, %v", ids, err)
	}
	if _, err := c.ListSiteResourceSiteIDs(context.Background(), "acme", 99); !IsNotFound(err) {
		t.Errorf("expected not found, got %v", err)
	}
}

func TestListSiteResourceRoleIDs_ExcludesAdmin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"roles": []any{
			map[string]any{"roleId": float64(1), "isAdmin": true},
			map[string]any{"roleId": float64(7), "isAdmin": false},
		}}, "success": true})
	}))
	defer server.Close()

	ids, err := New(server.URL, "token").ListSiteResourceRoleIDs(context.Background(), 12)
	if err != nil || len(ids) != 1 || ids[0] != 7 {
		t.Fatalf("unexpected ids: %v, %v", ids, err)
	}
}
