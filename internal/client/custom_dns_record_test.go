package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCustomDnsRecordLifecycle(t *testing.T) {
	record := map[string]any{"customDnsRecordId": float64(3), "domainId": "d1", "name": "mail", "fullDomain": "mail.example.com", "type": "MX", "values": []any{"mx.example.net"}, "ttl": float64(300), "priority": float64(10), "weight": nil, "port": nil, "enabled": true}
	result := map[string]any{"data": record, "success": true}
	list := map[string]any{"data": map[string]any{"records": []any{record}}, "success": true}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/org/acme/domain/d1/custom-dns-record":
			_ = json.NewEncoder(w).Encode(result)
		case r.Method == http.MethodGet && r.URL.Path == "/org/acme/domain/d1/custom-dns-records":
			_ = json.NewEncoder(w).Encode(list)
		case r.Method == http.MethodPost && r.URL.Path == "/org/acme/domain/d1/custom-dns-record/3":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			// A cleared field must reach the server as an explicit null;
			// omitting it would leave the old value in place.
			if value, sent := body["priority"]; !sent || value != nil {
				t.Errorf("expected explicit null priority, got %v (sent: %v)", value, sent)
			}
			_ = json.NewEncoder(w).Encode(result)
		case r.Method == http.MethodDelete && r.URL.Path == "/org/acme/domain/d1/custom-dns-record/3":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(server.URL, "token")
	ctx := context.Background()
	in := CustomDnsRecordRequest{Name: "mail", Type: "MX", Values: []string{"mx.example.net"}}
	created, err := c.CreateCustomDnsRecord(ctx, "acme", "d1", in)
	if err != nil || created.CustomDnsRecordID != 3 || created.Priority == nil || *created.Priority != 10 {
		t.Fatalf("create: %+v, %v", created, err)
	}
	got, err := c.GetCustomDnsRecord(ctx, "acme", "d1", 3)
	if err != nil || got.Weight != nil || len(got.Values) != 1 {
		t.Fatalf("get: %+v, %v", got, err)
	}
	if _, err := c.GetCustomDnsRecord(ctx, "acme", "d1", 4); !IsNotFound(err) {
		t.Fatalf("expected not found for an unlisted record, got %v", err)
	}
	if _, err := c.UpdateCustomDnsRecord(ctx, "acme", "d1", 3, in); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := c.DeleteCustomDnsRecord(ctx, "acme", "d1", 3); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
