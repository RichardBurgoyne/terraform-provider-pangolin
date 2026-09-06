package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddUserToResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/resource/5/users/add" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.AddUserToResource(context.Background(), 5, "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListResourceUsers_ContainsGrantedUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"users": []map[string]any{{"userId": "u1"}}},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	userIDs, err := c.ListResourceUsers(context.Background(), 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(userIDs) != 1 || userIDs[0] != "u1" {
		t.Errorf("expected [\"u1\"], got %v", userIDs)
	}
}

func TestRemoveUserFromResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/resource/5/users/remove" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.RemoveUserFromResource(context.Background(), 5, "u1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
