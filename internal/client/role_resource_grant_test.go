package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddRoleToResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/resource/5/roles/add" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.AddRoleToResource(context.Background(), 5, 3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListResourceRoles_ContainsGrantedRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"roles": []map[string]any{{"roleId": float64(3)}}},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	roleIDs, err := c.ListResourceRoles(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(roleIDs) != 1 || roleIDs[0] != 3 {
		t.Errorf("expected [3], got %v", roleIDs)
	}
}

func TestRemoveRoleFromResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/resource/5/roles/remove" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.RemoveRoleFromResource(context.Background(), 5, 3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
