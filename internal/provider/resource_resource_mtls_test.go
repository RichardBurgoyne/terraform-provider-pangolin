package provider

import (
	"encoding/pem"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPemCertificates(t *testing.T) {
	certificate := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("abc")}))
	key := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("nope")}))

	got := pemCertificates("# leading comment\n" + certificate + key)
	// SHA-256 of "abc".
	const fingerprint = "BA:78:16:BF:8F:01:CF:EA:41:41:40:DE:5D:AE:22:23:B0:03:61:A3:96:17:7A:9C:B4:10:FF:61:F2:00:15:AD"
	if len(got) != 1 || got[fingerprint] != certificate {
		t.Errorf("unexpected certificates: %v", got)
	}

	if got := pemCertificates("not pem"); len(got) != 0 {
		t.Errorf("expected no certificates, got %v", got)
	}
}

func TestResourceMtlsResource_ValidateConfigRejectsNonCertificate(t *testing.T) {
	model := resourceMtlsModel{
		ResourceID:       types.Int64Value(5),
		CACertificatePEM: types.StringValue("not pem"),
		Fingerprints:     types.SetNull(types.StringType),
	}
	if diags := validateConfig(t, &resourceMtlsResource{}, model); !diags.HasError() {
		t.Error("expected a validation error")
	}
}
