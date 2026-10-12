package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResourceRuleLifecycle(t *testing.T) {
	rule := map[string]any{"ruleId": float64(12), "resourceId": float64(5), "action": "DROP", "match": "METHOD", "value": "DELETE,PUT", "priority": float64(1), "enabled": true}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/resource/5/rule":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["match"] != "METHOD" || body["value"] != "DELETE,PUT" {
				t.Errorf("unexpected create body: %v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": rule, "success": true})
		case r.Method == http.MethodGet && r.URL.Path == "/resource/5/rules":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"rules": []any{rule}}, "success": true})
		case r.Method == http.MethodPost && r.URL.Path == "/resource/5/rule/12":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": rule, "success": true})
		case r.Method == http.MethodDelete && r.URL.Path == "/resource/5/rule/12":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": nil, "success": true})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	c := New(server.URL, "token")
	ctx := context.Background()
	created, err := c.CreateResourceRule(ctx, 5, CreateResourceRuleRequest{Action: "DROP", Match: "METHOD", Value: "DELETE,PUT", Priority: 1})
	if err != nil || created.RuleID != 12 {
		t.Fatalf("create: %+v, %v", created, err)
	}
	got, err := c.GetResourceRule(ctx, 5, 12)
	if err != nil || got.Match != "METHOD" || !got.Enabled {
		t.Fatalf("get: %+v, %v", got, err)
	}
	if _, err := c.GetResourceRule(ctx, 5, 99); !IsNotFound(err) {
		t.Errorf("expected not found for missing rule, got %v", err)
	}
	if _, err := c.UpdateResourceRule(ctx, 5, 12, UpdateResourceRuleRequest{Action: "DROP", Match: "METHOD", Value: "DELETE,PUT", Priority: 1}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := c.DeleteResourceRule(ctx, 5, 12); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
