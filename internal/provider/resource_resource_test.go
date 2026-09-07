package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// newTestConfig builds a tfsdk.Config for r from model, filling in any
// attribute not set on model as null. Used to exercise ValidateConfig
// directly without a full acceptance-test harness.
func newTestConfig(t *testing.T, r *pangolinResourceResource, model pangolinResourceModel) tfsdk.Config {
	t.Helper()
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	var value attr.Value
	diags := tfsdk.ValueFrom(ctx, model, schemaResp.Schema.Type(), &value)
	if diags.HasError() {
		t.Fatalf("building test config: %v", diags)
	}
	tfValue, err := value.ToTerraformValue(ctx)
	if err != nil {
		t.Fatalf("converting test config to terraform value: %v", err)
	}
	return tfsdk.Config{Raw: tfValue, Schema: schemaResp.Schema}
}

func TestPangolinResourceResource_Schema(t *testing.T) {
	r := NewPangolinResourceResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{
		"org_id", "resource_id", "name", "mode", "domain_id", "full_domain",
		"proxy_port", "proxy_protocol", "proxy_protocol_version", "ai_providers",
		"sso", "email_whitelist_enabled", "apply_rules", "skip_to_idp_id",
		"tls_server_name", "set_host_header", "headers_json",
	} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("expected attribute %q", name)
		}
	}

	// domain_id and proxy_port are mode-conditional (required for http-family
	// vs. raw tcp/udp resources respectively), so neither can be schema-level
	// Required; conditional requiredness is enforced in ValidateConfig instead.
	for _, name := range []string{"domain_id", "proxy_port"} {
		attr := resp.Schema.Attributes[name]
		if attr.IsRequired() {
			t.Errorf("expected %q not to be schema-level required", name)
		}
		if !attr.IsOptional() {
			t.Errorf("expected %q to be optional", name)
		}
	}
}

// Raw tcp/udp resources and http-family resources (http, ssh, rdp, vnc,
// inference) accept mutually exclusive sets of fields; the Pangolin API
// rejects unknown fields per mode with a strict schema. ValidateConfig is
// what catches a misconfigured resource before any API call is made.
func TestPangolinResourceResource_ValidateConfig_RawModeRejectsDomainID(t *testing.T) {
	r := &pangolinResourceResource{}
	config := pangolinResourceModel{
		Mode:        types.StringValue("tcp"),
		DomainID:    types.StringValue("d1"),
		ProxyPort:   types.Int64Value(5432),
		AIProviders: types.ListNull(aiProviderObjectType()),
	}

	resp := &resource.ValidateConfigResponse{}
	req := resource.ValidateConfigRequest{Config: newTestConfig(t, r, config)}
	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a validation error when domain_id is set on a tcp-mode resource")
	}
}

func TestPangolinResourceResource_ValidateConfig_RawModeRequiresProxyPort(t *testing.T) {
	r := &pangolinResourceResource{}
	config := pangolinResourceModel{
		Mode:        types.StringValue("udp"),
		AIProviders: types.ListNull(aiProviderObjectType()),
	}

	resp := &resource.ValidateConfigResponse{}
	req := resource.ValidateConfigRequest{Config: newTestConfig(t, r, config)}
	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a validation error when proxy_port is unset on a udp-mode resource")
	}
}

func TestPangolinResourceResource_ValidateConfig_HTTPModeRequiresDomainID(t *testing.T) {
	r := &pangolinResourceResource{}
	config := pangolinResourceModel{
		Mode:        types.StringValue("http"),
		AIProviders: types.ListNull(aiProviderObjectType()),
	}

	resp := &resource.ValidateConfigResponse{}
	req := resource.ValidateConfigRequest{Config: newTestConfig(t, r, config)}
	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a validation error when domain_id is unset on an http-mode resource")
	}
}

func TestPangolinResourceResource_ValidateConfig_RawModeRejectsSSO(t *testing.T) {
	r := &pangolinResourceResource{}
	config := pangolinResourceModel{
		Mode:        types.StringValue("tcp"),
		ProxyPort:   types.Int64Value(5432),
		SSO:         types.BoolValue(true),
		AIProviders: types.ListNull(aiProviderObjectType()),
	}

	resp := &resource.ValidateConfigResponse{}
	req := resource.ValidateConfigRequest{Config: newTestConfig(t, r, config)}
	r.ValidateConfig(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a validation error when sso is set on a tcp-mode resource")
	}
}

func TestPangolinResourceResource_ValidateConfig_ValidHTTPConfig(t *testing.T) {
	r := &pangolinResourceResource{}
	config := pangolinResourceModel{
		Mode:        types.StringValue("http"),
		DomainID:    types.StringValue("d1"),
		AIProviders: types.ListNull(aiProviderObjectType()),
	}

	resp := &resource.ValidateConfigResponse{}
	req := resource.ValidateConfigRequest{Config: newTestConfig(t, r, config)}
	r.ValidateConfig(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected validation error: %v", resp.Diagnostics)
	}
}

// Regression test: post_auth_path, pam_mode and auth_daemon_mode are written
// unconditionally from the API response, which returns "" when the field is
// absent (e.g. mode = http). If they were Optional without Computed, that ""
// would conflict with a null config value and apply would fail with "Provider
// produced inconsistent result after apply".
func TestPangolinResourceResource_OptionalComputedAttributes(t *testing.T) {
	r := NewPangolinResourceResource()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

	for _, name := range []string{"post_auth_path", "pam_mode", "auth_daemon_mode"} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Errorf("expected attribute %q", name)
			continue
		}
		if !attr.IsOptional() {
			t.Errorf("expected %q to be optional", name)
		}
		if !attr.IsComputed() {
			t.Errorf("expected %q to be computed (it is set unconditionally from the API response)", name)
		}
	}
}
