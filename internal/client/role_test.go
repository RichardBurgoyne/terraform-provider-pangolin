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
		_ = json.NewEncoder(w).Encode(map[string]any{
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
		_ = json.NewEncoder(w).Encode(map[string]any{
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

func TestCreateRole_SendsSSHSudoFieldsAsPlainArrays(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"roleId":           float64(3),
				"orgId":            "acme",
				"name":             "Editors",
				"sshSudoMode":      "commands",
				"sshSudoCommands":  `["/usr/bin/systemctl restart nginx"]`,
				"sshCreateHomeDir": true,
				"sshUnixGroups":    `["docker"]`,
			},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	homeDir := true
	role, err := c.CreateRole(context.Background(), "acme", CreateRoleRequest{
		Name:             "Editors",
		SSHSudoMode:      "commands",
		SSHSudoCommands:  []string{"/usr/bin/systemctl restart nginx"},
		SSHCreateHomeDir: &homeDir,
		SSHUnixGroups:    []string{"docker"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The request body must send a plain JSON array, not a JSON-encoded string.
	if cmds, ok := received["sshSudoCommands"].([]any); !ok || len(cmds) != 1 || cmds[0] != "/usr/bin/systemctl restart nginx" {
		t.Errorf("expected sshSudoCommands to be sent as a plain array, got %#v", received["sshSudoCommands"])
	}

	// The response echoes back a JSON-encoded string, which must decode cleanly.
	commands, err := DecodeSSHStringList(role.SSHSudoCommands)
	if err != nil {
		t.Fatalf("unexpected error decoding sshSudoCommands: %v", err)
	}
	if len(commands) != 1 || commands[0] != "/usr/bin/systemctl restart nginx" {
		t.Errorf("unexpected decoded sshSudoCommands: %#v", commands)
	}

	groups, err := DecodeSSHStringList(role.SSHUnixGroups)
	if err != nil {
		t.Fatalf("unexpected error decoding sshUnixGroups: %v", err)
	}
	if len(groups) != 1 || groups[0] != "docker" {
		t.Errorf("unexpected decoded sshUnixGroups: %#v", groups)
	}
}

func TestDecodeSSHStringList_Empty(t *testing.T) {
	list, err := DecodeSSHStringList("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list != nil {
		t.Errorf("expected nil list for empty string, got %#v", list)
	}

	list, err = DecodeSSHStringList("[]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %#v", list)
	}
}

func TestDeleteRole(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/role/3" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteRole(context.Background(), 3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
