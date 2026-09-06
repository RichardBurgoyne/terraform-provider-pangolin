package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/org/acme/role" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"roleId": float64(3), "orgId": "acme", "name": "Editors"},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	role, err := c.CreateRole(context.Background(), "acme", CreateRoleRequest{Name: "Editors"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role.RoleID != 3 {
		t.Errorf("expected roleId 3, got %d", role.RoleID)
	}
}

func TestGetRoleByID_FiltersListRoles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/org/acme/roles" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"roles": []map[string]any{
					{"roleId": float64(3), "orgId": "acme", "name": "Editors"},
					{"roleId": float64(4), "orgId": "acme", "name": "Viewers"},
				},
			},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	role, err := c.GetRoleByID(context.Background(), "acme", 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if role.Name != "Viewers" {
		t.Errorf("expected role 'Viewers', got %+v", role)
	}

	if _, err := c.GetRoleByID(context.Background(), "acme", 999); !IsNotFound(err) {
		t.Errorf("expected IsNotFound for missing role, got %v", err)
	}
}

func TestDeleteRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/role/3" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteRole(context.Background(), 3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
