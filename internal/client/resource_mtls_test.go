package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResourceMtlsCertificates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/resource/5/mtls/certificate":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["certificate"] != "PEM" {
				t.Errorf("unexpected body: %v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"certificates": []any{}}, "success": true})
		case r.Method == http.MethodGet && r.URL.Path == "/resource/5/mtls/certificates":
			certs := []any{map[string]any{"mtlsCertificateId": 2, "subject": "CN=ca", "fingerprint": "AA:BB"}}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"certificates": certs}, "success": true})
		case r.Method == http.MethodDelete && r.URL.Path == "/resource/5/mtls/certificate/2":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"mtlsDisabled": true}, "success": true})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(server.URL, "token")
	ctx := context.Background()
	if err := c.AddResourceMtlsCertificates(ctx, 5, "PEM"); err != nil {
		t.Fatalf("add: %v", err)
	}
	certs, err := c.ListResourceMtlsCertificates(ctx, 5)
	if err != nil || len(certs) != 1 || certs[0].Fingerprint != "AA:BB" {
		t.Fatalf("list: %+v, %v", certs, err)
	}
	if err := c.DeleteResourceMtlsCertificate(ctx, 5, 2); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
