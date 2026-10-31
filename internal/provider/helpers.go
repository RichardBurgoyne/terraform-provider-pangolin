package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

// resourceClient is embedded by every resource to supply the API client and
// the Configure method that sets it.
type resourceClient struct {
	client *client.Client
}

func (r *resourceClient) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	r.client = c
}

// dataSourceClient is the data source counterpart of resourceClient.
type dataSourceClient struct {
	client *client.Client
}

func (d *dataSourceClient) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

// splitImportID splits a two-part `<first>:<second>` import ID.
func splitImportID(id, firstAttr, secondAttr, example string, diags *diag.Diagnostics) (string, string, bool) {
	first, second, ok := strings.Cut(id, ":")
	if !ok || first == "" || second == "" {
		diags.AddError("Invalid Import ID", fmt.Sprintf(`expected format: <%s>:<%s>, e.g. %q`, firstAttr, secondAttr, example))
		return "", "", false
	}
	return first, second, true
}

func parseImportInt64(attr, value string, diags *diag.Diagnostics) (int64, bool) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		diags.AddError("Invalid Import ID", attr+" must be a numeric ID: "+err.Error())
		return 0, false
	}
	return id, true
}

// importStringAndInt64 imports a `<string>:<int64>` ID, such as
// `<org_id>:<resource_id>`.
func importStringAndInt64(ctx context.Context, id string, resp *resource.ImportStateResponse, firstAttr, secondAttr string) {
	first, second, ok := splitImportID(id, firstAttr, secondAttr, "acme:5", &resp.Diagnostics)
	if !ok {
		return
	}
	secondID, ok := parseImportInt64(secondAttr, second, &resp.Diagnostics)
	if !ok {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(firstAttr), first)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(secondAttr), secondID)...)
}

// importInt64Pair imports an `<int64>:<int64>` ID, such as
// `<resource_id>:<target_id>`.
func importInt64Pair(ctx context.Context, id string, resp *resource.ImportStateResponse, firstAttr, secondAttr string) {
	first, second, ok := splitImportID(id, firstAttr, secondAttr, "5:9", &resp.Diagnostics)
	if !ok {
		return
	}
	firstID, ok := parseImportInt64(firstAttr, first, &resp.Diagnostics)
	if !ok {
		return
	}
	secondID, ok := parseImportInt64(secondAttr, second, &resp.Diagnostics)
	if !ok {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(firstAttr), firstID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(secondAttr), secondID)...)
}

// importStringPair imports a `<string>:<string>` ID, such as
// `<org_id>:<domain_id>`.
func importStringPair(ctx context.Context, id string, resp *resource.ImportStateResponse, firstAttr, secondAttr string) {
	first, second, ok := splitImportID(id, firstAttr, secondAttr, "acme:d1", &resp.Diagnostics)
	if !ok {
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(firstAttr), first)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(secondAttr), second)...)
}

// known reports whether a config or plan value is neither null nor unknown.
func known[T interface {
	IsNull() bool
	IsUnknown() bool
}](v T) bool {
	return !v.IsNull() && !v.IsUnknown()
}

func strPtr(v types.String) *string {
	if !known(v) {
		return nil
	}
	s := v.ValueString()
	return &s
}

func int64Ptr(v types.Int64) *int64 {
	if !known(v) {
		return nil
	}
	i := v.ValueInt64()
	return &i
}

func boolPtr(v types.Bool) *bool {
	if !known(v) {
		return nil
	}
	b := v.ValueBool()
	return &b
}

func setToInt64s(ctx context.Context, s types.Set) ([]int64, diag.Diagnostics) {
	out := []int64{}
	if !known(s) {
		return out, nil
	}
	d := s.ElementsAs(ctx, &out, false)
	return out, d
}

func setToStrings(ctx context.Context, s types.Set) ([]string, diag.Diagnostics) {
	out := []string{}
	if !known(s) {
		return out, nil
	}
	d := s.ElementsAs(ctx, &out, false)
	return out, d
}

func nullableString(p *string) types.String {
	if p == nil {
		return types.StringNull()
	}
	return types.StringValue(*p)
}

func nullableInt64(p *int64) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*p)
}

// nonZeroInt64 maps the API's "unset" zero value to null.
func nonZeroInt64(v int64) types.Int64 {
	if v == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(v)
}

// nonEmptyString maps the API's "unset" empty value to null.
func nonEmptyString(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}
