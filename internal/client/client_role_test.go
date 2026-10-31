package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestClientRoles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/client/9":
			roles := []any{map[string]any{"roleId": 2, "name": "ops"}}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"roles": roles}, "success": true})
		case r.Method == http.MethodPost && r.URL.Path == "/client/9/add-role/2",
			r.Method == http.MethodDelete && r.URL.Path == "/client/9/remove-role/2":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(server.URL, "token")
	ctx := context.Background()
	if err := c.AddClientRole(ctx, 9, 2); err != nil {
		t.Fatalf("add: %v", err)
	}
	ids, err := c.ListClientRoleIDs(ctx, 9)
	if err != nil || !slices.Equal(ids, []int64{2}) {
		t.Fatalf("list: %v, %v", ids, err)
	}
	if err := c.RemoveClientRole(ctx, 9, 2); err != nil {
		t.Fatalf("remove: %v", err)
	}
}
