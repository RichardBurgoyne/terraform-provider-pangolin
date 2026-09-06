// internal/client/organization_test.go
package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateOrganization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/org" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"orgId": "acme", "name": "Acme", "subnet": "10.0.0.0/24", "utilitySubnet": "10.0.1.0/24",
			},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	org, err := c.CreateOrganization(context.Background(), CreateOrganizationRequest{
		OrgID: "acme", Name: "Acme", Subnet: "10.0.0.0/24", UtilitySubnet: "10.0.1.0/24",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if org.OrgID != "acme" || org.Name != "Acme" {
		t.Errorf("unexpected organization: %+v", org)
	}
}

func TestGetOrganization_UnwrapsOrgWrapper(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/org/acme" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"org": map[string]any{
					"orgId": "acme", "name": "Acme", "subnet": "10.0.0.0/24", "utilitySubnet": "10.0.1.0/24",
				},
			},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	org, err := c.GetOrganization(context.Background(), "acme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if org.OrgID != "acme" {
		t.Errorf("expected orgId 'acme', got %+v", org)
	}
}

func TestDeleteOrganization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/org/acme" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteOrganization(context.Background(), "acme"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
