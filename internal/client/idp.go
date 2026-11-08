package client

import (
	"context"
	"fmt"
	"net/http"
)

// OidcIdp is a global OIDC identity provider, flattened from the API's
// separate idp and idpOidcConfig objects. Nullable columns come back as JSON
// null, hence the pointers.
type OidcIdp struct {
	IdpID               int64
	Name                string
	DefaultRoleMapping  *string
	DefaultOrgMapping   *string
	AutoProvision       bool
	SessionRevalidation bool
	Tags                *string
	Variant             string
	ClientID            string
	ClientSecret        string
	AuthURL             string
	TokenURL            string
	IdentifierPath      string
	EmailPath           *string
	NamePath            *string
	Scopes              string
}

// OidcIdpRequest is the body for both create and update. Create rejects
// DefaultRoleMapping and DefaultOrgMapping, so they are only sent on update.
// SessionRevalidation is only understood by Pangolin 1.26 and later.
type OidcIdpRequest struct {
	Name                string  `json:"name"`
	ClientID            string  `json:"clientId"`
	ClientSecret        string  `json:"clientSecret"`
	AuthURL             string  `json:"authUrl"`
	TokenURL            string  `json:"tokenUrl"`
	IdentifierPath      string  `json:"identifierPath"`
	EmailPath           *string `json:"emailPath,omitempty"`
	NamePath            *string `json:"namePath,omitempty"`
	Scopes              string  `json:"scopes"`
	AutoProvision       *bool   `json:"autoProvision,omitempty"`
	SessionRevalidation *bool   `json:"sessionRevalidation,omitempty"`
	Tags                *string `json:"tags,omitempty"`
	Variant             *string `json:"variant,omitempty"`
	DefaultRoleMapping  *string `json:"defaultRoleMapping,omitempty"`
	DefaultOrgMapping   *string `json:"defaultOrgMapping,omitempty"`
}

type idpResult struct {
	Idp struct {
		IdpID               int64   `json:"idpId"`
		Name                string  `json:"name"`
		Type                string  `json:"type"`
		DefaultRoleMapping  *string `json:"defaultRoleMapping"`
		DefaultOrgMapping   *string `json:"defaultOrgMapping"`
		AutoProvision       bool    `json:"autoProvision"`
		SessionRevalidation bool    `json:"sessionRevalidation"`
		Tags                *string `json:"tags"`
	} `json:"idp"`
	OidcConfig *struct {
		Variant        string  `json:"variant"`
		ClientID       string  `json:"clientId"`
		ClientSecret   string  `json:"clientSecret"`
		AuthURL        string  `json:"authUrl"`
		TokenURL       string  `json:"tokenUrl"`
		IdentifierPath string  `json:"identifierPath"`
		EmailPath      *string `json:"emailPath"`
		NamePath       *string `json:"namePath"`
		Scopes         string  `json:"scopes"`
	} `json:"idpOidcConfig"`
}

// CreateOidcIdp creates a global OIDC identity provider and returns its ID
// and the redirect URL to register with the identity provider.
func (c *Client) CreateOidcIdp(ctx context.Context, in OidcIdpRequest) (int64, string, error) {
	in.DefaultRoleMapping, in.DefaultOrgMapping = nil, nil
	var out struct {
		IdpID       int64  `json:"idpId"`
		RedirectURL string `json:"redirectUrl"`
	}
	if err := c.do(ctx, http.MethodPut, "/idp/oidc", in, &out); err != nil {
		return 0, "", err
	}
	return out.IdpID, out.RedirectURL, nil
}

func (c *Client) GetOidcIdp(ctx context.Context, idpID int64) (*OidcIdp, error) {
	var out idpResult
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/idp/%d", idpID), nil, &out); err != nil {
		return nil, err
	}
	if out.Idp.Type != "oidc" || out.OidcConfig == nil {
		return nil, fmt.Errorf("identity provider %d is of type %q, not oidc", idpID, out.Idp.Type)
	}
	return &OidcIdp{
		IdpID:               out.Idp.IdpID,
		Name:                out.Idp.Name,
		DefaultRoleMapping:  out.Idp.DefaultRoleMapping,
		DefaultOrgMapping:   out.Idp.DefaultOrgMapping,
		AutoProvision:       out.Idp.AutoProvision,
		SessionRevalidation: out.Idp.SessionRevalidation,
		Tags:                out.Idp.Tags,
		Variant:             out.OidcConfig.Variant,
		ClientID:            out.OidcConfig.ClientID,
		ClientSecret:        out.OidcConfig.ClientSecret,
		AuthURL:             out.OidcConfig.AuthURL,
		TokenURL:            out.OidcConfig.TokenURL,
		IdentifierPath:      out.OidcConfig.IdentifierPath,
		EmailPath:           out.OidcConfig.EmailPath,
		NamePath:            out.OidcConfig.NamePath,
		Scopes:              out.OidcConfig.Scopes,
	}, nil
}

func (c *Client) UpdateOidcIdp(ctx context.Context, idpID int64, in OidcIdpRequest) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/idp/%d/oidc", idpID), in, nil)
}

func (c *Client) DeleteIdp(ctx context.Context, idpID int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/idp/%d", idpID), nil, nil)
}
