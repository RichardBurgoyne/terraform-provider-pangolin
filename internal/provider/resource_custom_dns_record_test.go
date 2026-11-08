package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/RichardBurgoyne/terraform-provider-pangolin/internal/client"
)

func TestCustomDnsRecordResource_ValidateConfig(t *testing.T) {
	values := func(v ...string) types.List {
		list, _ := types.ListValueFrom(context.Background(), types.StringType, v)
		return list
	}
	for name, tc := range map[string]struct {
		model   customDnsRecordModel
		wantErr bool
	}{
		"A":                   {customDnsRecordModel{Type: types.StringValue("A"), Values: values("192.0.2.1")}, false},
		"A with priority":     {customDnsRecordModel{Type: types.StringValue("A"), Values: values("192.0.2.1"), Priority: types.Int64Value(10)}, true},
		"MX":                  {customDnsRecordModel{Type: types.StringValue("MX"), Values: values("mx.example.net"), Priority: types.Int64Value(10)}, false},
		"MX without priority": {customDnsRecordModel{Type: types.StringValue("MX"), Values: values("mx.example.net")}, true},
		"MX with port":        {customDnsRecordModel{Type: types.StringValue("MX"), Values: values("mx.example.net"), Priority: types.Int64Value(10), Port: types.Int64Value(25)}, true},
		"SRV": {customDnsRecordModel{Type: types.StringValue("SRV"), Values: values("sip.example.net"),
			Priority: types.Int64Value(10), Weight: types.Int64Value(5), Port: types.Int64Value(5060)}, false},
		"SRV without weight": {customDnsRecordModel{Type: types.StringValue("SRV"), Values: values("sip.example.net"),
			Priority: types.Int64Value(10), Port: types.Int64Value(5060)}, true},
		"CNAME with two values": {customDnsRecordModel{Type: types.StringValue("CNAME"), Values: values("a.example.net", "b.example.net")}, true},
	} {
		t.Run(name, func(t *testing.T) {
			tc.model.Name = types.StringValue("www")
			diags := validateConfig(t, &customDnsRecordResource{}, tc.model)
			if diags.HasError() != tc.wantErr {
				t.Errorf("wantErr %v, got %v", tc.wantErr, diags)
			}
		})
	}
}

func TestNormalizeDnsName(t *testing.T) {
	for in, want := range map[string]string{
		"":                    "@",
		"@":                   "@",
		"Example.com.":        "@",
		"WWW":                 "www",
		"www.example.com":     "www",
		"_dmarc.example.com.": "_dmarc",
		"*.apps":              "*.apps",
	} {
		if got := normalizeDnsName(in, "example.com"); got != want {
			t.Errorf("normalizeDnsName(%q) = %q, want %q", in, got, want)
		}
	}
}

// Pangolin normalizes what it stores; the configured spelling must survive a
// refresh so the plan stays empty.
func TestCustomDnsRecordModel_FromAPIKeepsConfiguredSpelling(t *testing.T) {
	ctx := context.Background()
	configured, _ := types.ListValueFrom(ctx, types.StringType, []string{"MX.Example.NET."})
	model := customDnsRecordModel{Name: types.StringValue("Mail.example.com"), Values: configured}
	priority := int64(10)
	diags := model.fromAPI(ctx, &client.CustomDnsRecord{
		CustomDnsRecordID: 3, DomainID: "d1", Name: "mail", FullDomain: "mail.example.com", Type: "MX",
		Values: []string{"mx.example.net"}, TTL: 300, Priority: &priority, Enabled: true,
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	if model.Name.ValueString() != "Mail.example.com" || !model.Values.Equal(configured) {
		t.Errorf("configured spelling was replaced: %s %s", model.Name, model.Values)
	}

	// A real change on the server still shows up.
	diags = model.fromAPI(ctx, &client.CustomDnsRecord{
		CustomDnsRecordID: 3, DomainID: "d1", Name: "smtp", FullDomain: "smtp.example.com", Type: "MX",
		Values: []string{"mx2.example.net"}, TTL: 300, Priority: &priority, Enabled: true,
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	var values []string
	model.Values.ElementsAs(ctx, &values, false)
	if model.Name.ValueString() != "smtp" || len(values) != 1 || values[0] != "mx2.example.net" {
		t.Errorf("server change not picked up: %s %v", model.Name, values)
	}
}

func TestNormalizeDnsValue_TXTQuotes(t *testing.T) {
	if got := normalizeDnsValue("TXT", `"v=spf1 -all"`); got != "v=spf1 -all" {
		t.Errorf("got %q", got)
	}
	if got := normalizeDnsValue("TXT", "Case Kept"); got != "Case Kept" {
		t.Errorf("got %q", got)
	}
}
