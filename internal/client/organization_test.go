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
		_ = json.NewEncoder(w).Encode(map[string]any{
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
		_ = json.NewEncoder(w).Encode(map[string]any{
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
		_ = json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteOrganization(context.Background(), "acme"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateOrganization_SendsOnlySetFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/org/acme" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		// An older server rejects fields it doesn't know, so unset settings
		// must not be sent at all.
		if len(body) != 2 || body["name"] != "Acme Ltd" || body["settingsLogRetentionDaysSsh"] != float64(30) {
			t.Errorf("unexpected body: %v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	name, days := "Acme Ltd", int64(30)
	c := New(server.URL, "token")
	if err := c.UpdateOrganization(context.Background(), "acme", UpdateOrganizationRequest{Name: &name, LogRetentionDaysSSH: &days}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
