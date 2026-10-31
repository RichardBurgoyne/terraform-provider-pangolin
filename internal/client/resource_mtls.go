package client

import (
	"context"
	"fmt"
	"net/http"
)

// MtlsCertificate is a trusted CA on a resource. The API never returns the
// PEM itself. Fingerprint is the colon-separated, upper-case SHA-256 of the
// certificate's DER encoding.
type MtlsCertificate struct {
	MtlsCertificateID int64  `json:"mtlsCertificateId"`
	Subject           string `json:"subject"`
	Fingerprint       string `json:"fingerprint"`
}

type addMtlsCertificateRequest struct {
	Certificate string `json:"certificate"`
}

func (c *Client) ListResourceMtlsCertificates(ctx context.Context, resourceID int64) ([]MtlsCertificate, error) {
	var out struct {
		Certificates []MtlsCertificate `json:"certificates"`
	}
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/resource/%d/mtls/certificates", resourceID), nil, &out); err != nil {
		return nil, err
	}
	return out.Certificates, nil
}

// AddResourceMtlsCertificates adds the certificates in a PEM bundle. The API
// fails with a 409 if every certificate in the bundle is already present.
func (c *Client) AddResourceMtlsCertificates(ctx context.Context, resourceID int64, pem string) error {
	return c.do(ctx, http.MethodPut, fmt.Sprintf("/resource/%d/mtls/certificate", resourceID), addMtlsCertificateRequest{Certificate: pem}, nil)
}

// DeleteResourceMtlsCertificate removes a trusted CA. Removing the last one
// also turns mTLS off on the resource.
func (c *Client) DeleteResourceMtlsCertificate(ctx context.Context, resourceID, certificateID int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/resource/%d/mtls/certificate/%d", resourceID, certificateID), nil, nil)
}
