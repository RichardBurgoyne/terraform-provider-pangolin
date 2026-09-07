package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/resource/5/target" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":    map[string]any{"targetId": float64(9), "siteId": float64(1), "ip": "10.0.0.5", "port": float64(80), "enabled": true},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	target, err := c.CreateTarget(context.Background(), 5, CreateTargetRequest{SiteID: 1, IP: "10.0.0.5", Port: 80})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if target.TargetID != 9 {
		t.Errorf("expected targetId 9, got %d", target.TargetID)
	}
}

func TestCreateTarget_HealthCheckFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"targetId": float64(9), "siteId": float64(1), "ip": "10.0.0.5", "port": float64(80), "enabled": true,
				"hcEnabled": true, "hcHostname": "10.0.0.5", "hcPath": "/healthz", "hcInterval": float64(30),
				"hcHealth": "unhealthy",
				// create/update responses echo the raw db column: a JSON-encoded string.
				"hcHeaders": `[{"name":"X-Check","value":"1"}]`,
			},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	hcEnabled := true
	hcHostname := "10.0.0.5"
	target, err := c.CreateTarget(context.Background(), 5, CreateTargetRequest{
		SiteID: 1, IP: "10.0.0.5", Port: 80,
		HCEnabled:  &hcEnabled,
		HCHostname: &hcHostname,
		HCHeaders:  []HCHeader{{Name: "X-Check", Value: "1"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !target.HCEnabled || target.HCHealth != "unhealthy" {
		t.Errorf("unexpected target health check state: %+v", target)
	}

	headers, err := DecodeHCHeaders(target.HCHeaders)
	if err != nil {
		t.Fatalf("unexpected error decoding hcHeaders: %v", err)
	}
	if len(headers) != 1 || headers[0].Name != "X-Check" || headers[0].Value != "1" {
		t.Errorf("unexpected decoded hcHeaders: %#v", headers)
	}
}

func TestGetTarget_HealthCheckHeadersAsArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"targetId": float64(9), "siteId": float64(1), "ip": "10.0.0.5", "port": float64(80),
				// GET responses parse hcHeaders back into a real array.
				"hcHeaders": []map[string]any{{"name": "X-Check", "value": "1"}},
			},
			"success": true,
		})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	target, err := c.GetTarget(context.Background(), 9)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	headers, err := DecodeHCHeaders(target.HCHeaders)
	if err != nil {
		t.Fatalf("unexpected error decoding hcHeaders: %v", err)
	}
	if len(headers) != 1 || headers[0].Name != "X-Check" || headers[0].Value != "1" {
		t.Errorf("unexpected decoded hcHeaders: %#v", headers)
	}
}

func TestDeleteTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/target/9" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
	}))
	defer server.Close()

	c := New(server.URL, "token")
	if err := c.DeleteTarget(context.Background(), 9); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
