package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateOrgUser_ThenLookUpByUsername(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/org/acme/user":
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}, "success": true})
		case r.Method == http.MethodGet && r.URL.Path == "/org/acme/users":
			json.NewEncoder(w).Encode(map[string]any{
				"data":    map[string]any{"users": []map[string]any{{"userId": "u1", "username": "jane", "type": "oidc"}}},
				"success": true,
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.CreateOrgUser(context.Background(), "acme", CreateOrgUserRequest{Username: "jane", Type: "oidc", IdpID: 1, RoleIDs: []int64{2}}); err != nil {
		t.Fatalf("unexpected error creating user: %v", err)
	}

	user, err := c.GetUserByUsername(context.Background(), "acme", "jane")
	if err != nil {
		t.Fatalf("unexpected error looking up user: %v", err)
	}
	if user.UserID != "u1" {
		t.Errorf("expected userId 'u1', got %q", user.UserID)
	}
}

func TestAddUserRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/role/2/add/u1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.AddUserRole(context.Background(), 2, "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteOrgUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/org/acme/user/u1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteOrgUser(context.Background(), "acme", "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
