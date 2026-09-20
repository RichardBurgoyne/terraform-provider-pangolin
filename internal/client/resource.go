package client

import (
	"context"
	"fmt"
	"net/http"
)

type PangolinResource struct {
	ResourceID           int64  `json:"resourceId"`
	NiceID               string `json:"niceId"`
	Name                 string `json:"name"`
	Mode                 string `json:"mode"`
	DomainID             string `json:"domainId"`
	FullDomain           string `json:"fullDomain"`
	Subdomain            string `json:"subdomain"`
	StickySession        bool   `json:"stickySession"`
	PostAuthPath         string `json:"postAuthPath"`
	PamMode              string `json:"pamMode"`
	AuthDaemonMode       string `json:"authDaemonMode"`
	AuthDaemonPort       int64  `json:"authDaemonPort"`
	Enabled              bool   `json:"enabled"`
	SSL                  bool   `json:"ssl"`
	ProxyPort            int64  `json:"proxyPort"`
	ProxyProtocol        bool   `json:"proxyProtocol"`
	ProxyProtocolVersion int64  `json:"proxyProtocolVersion"`

	// The following are update-only fields (not settable at create time) and,
	// for SSO/EmailWhitelistEnabled/ApplyRules/SkipToIdpID, are actually
	// stored on the resource's auto-created default policy rather than the
	// resource itself. GetResource and UpdateResource both merge the policy's
	// values into the response under these same field names; CreateResource's
	// response does not (it reports them as their zero value regardless of
	// the policy), so a resource must be read or updated at least once after
	// creation to see their true values. They don't apply to raw tcp/udp
	// resources.
	SSO                   bool   `json:"sso"`
	EmailWhitelistEnabled bool   `json:"emailWhitelistEnabled"`
	ApplyRules            bool   `json:"applyRules"`
	SkipToIdpID           int64  `json:"skipToIdpId"`
	TLSServerName         string `json:"tlsServerName"`
	SetHostHeader         string `json:"setHostHeader"`

	// Headers is a JSON array of {name, value} objects. Like Target.HCHeaders,
	// the API is inconsistent about its wire format: GET returns a real JSON
	// array, but the create/update responses return the raw database column,
	// a JSON-encoded string. Decode with DecodeHCHeaders (shared with Target,
	// since both fields have identical shape and the same quirk).
	Headers any `json:"headers"`
}

type CreateResourceRequest struct {
	Name           string `json:"name"`
	Mode           string `json:"mode"`
	DomainID       string `json:"domainId,omitempty"`
	Subdomain      string `json:"subdomain,omitempty"`
	StickySession  *bool  `json:"stickySession,omitempty"`
	PostAuthPath   string `json:"postAuthPath,omitempty"`
	PamMode        string `json:"pamMode,omitempty"`
	AuthDaemonMode string `json:"authDaemonMode,omitempty"`
	AuthDaemonPort *int64 `json:"authDaemonPort,omitempty"`

	// ProxyPort is required for, and only valid on, raw tcp/udp resources.
	// Its presence is what the Pangolin API uses to route the request to the
	// raw-resource code path instead of the http-family one, so it must never
	// be set alongside DomainID.
	ProxyPort *int64 `json:"proxyPort,omitempty"`

	// AIProviders attaches AI providers (already configured in Pangolin) to
	// an inference-mode resource at creation time. Only valid when
	// Mode is "inference".
	AIProviders []ResourceAIProviderAttachment `json:"aiProviders,omitempty"`
}

type UpdateResourceRequest struct {
	Name          *string `json:"name,omitempty"`
	NiceID        *string `json:"niceId,omitempty"`
	Enabled       *bool   `json:"enabled,omitempty"`
	StickySession *bool   `json:"stickySession,omitempty"`

	// The following fields are only accepted by the API for http-family
	// resources (http, ssh, rdp, vnc, inference) and must be left nil when
	// updating a raw tcp/udp resource: the server validates the update body
	// with a strict schema per resource mode and rejects unknown fields.
	Subdomain      *string `json:"subdomain,omitempty"`
	SSL            *bool   `json:"ssl,omitempty"`
	DomainID       *string `json:"domainId,omitempty"`
	PostAuthPath   *string `json:"postAuthPath,omitempty"`
	PamMode        *string `json:"pamMode,omitempty"`
	AuthDaemonMode *string `json:"authDaemonMode,omitempty"`
	AuthDaemonPort *int64  `json:"authDaemonPort,omitempty"`

	// The following fields are only accepted by the API for raw tcp/udp
	// resources and must be left nil when updating an http-family resource.
	ProxyPort            *int64 `json:"proxyPort,omitempty"`
	ProxyProtocol        *bool  `json:"proxyProtocol,omitempty"`
	ProxyProtocolVersion *int64 `json:"proxyProtocolVersion,omitempty"`

	// The following are additional http-family-only fields (see the same
	// comment on PangolinResource for the default-policy indirection quirk
	// affecting SSO/EmailWhitelistEnabled/ApplyRules/SkipToIdpID).
	SSO                   *bool      `json:"sso,omitempty"`
	EmailWhitelistEnabled *bool      `json:"emailWhitelistEnabled,omitempty"`
	ApplyRules            *bool      `json:"applyRules,omitempty"`
	SkipToIdpID           *int64     `json:"skipToIdpId,omitempty"`
	TLSServerName         *string    `json:"tlsServerName,omitempty"`
	SetHostHeader         *string    `json:"setHostHeader,omitempty"`
	Headers               []HCHeader `json:"headers,omitempty"`
}

// ResourceAIProviderAttachment attaches an AI provider (already configured in
// Pangolin, outside this provider's scope) to an inference-mode resource.
type ResourceAIProviderAttachment struct {
	ProviderID int64  `json:"providerId"`
	AccessMode string `json:"accessMode,omitempty"` // inherit (default) or select
	Enabled    *bool  `json:"enabled,omitempty"`
}

// ResourceAIProvider is an AI provider attachment as returned by
// ListResourceAIProviders, including read-only details about the provider
// itself.
type ResourceAIProvider struct {
	ProviderID      int64  `json:"providerId"`
	NiceID          string `json:"niceId"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	Enabled         bool   `json:"enabled"`
	ProviderEnabled bool   `json:"providerEnabled"`
	AccessMode      string `json:"accessMode"`
}

type setResourceAIProvidersRequest struct {
	Providers []ResourceAIProviderAttachment `json:"providers"`
}

type listResourceAIProvidersResult struct {
	Providers []ResourceAIProvider `json:"providers"`
}

func (c *Client) CreateResource(ctx context.Context, orgID string, in CreateResourceRequest) (*PangolinResource, error) {
	var out PangolinResource
	if err := c.do(ctx, http.MethodPut, "/org/"+pathEscape(orgID)+"/resource", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetResource(ctx context.Context, resourceID int64) (*PangolinResource, error) {
	var out PangolinResource
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/resource/%d", resourceID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateResource(ctx context.Context, resourceID int64, in UpdateResourceRequest) (*PangolinResource, error) {
	var out PangolinResource
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/resource/%d", resourceID), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteResource(ctx context.Context, resourceID int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/resource/%d", resourceID), nil, nil)
}

// SetResourceAIProviders replaces the full set of AI providers attached to an
// inference-mode resource. An empty slice clears all attachments.
func (c *Client) SetResourceAIProviders(ctx context.Context, resourceID int64, providers []ResourceAIProviderAttachment) error {
	if providers == nil {
		providers = []ResourceAIProviderAttachment{}
	}
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/resource/%d/ai-providers", resourceID), setResourceAIProvidersRequest{Providers: providers}, nil)
}

// ListResourceAIProviders lists the AI providers currently attached to an
// inference-mode resource.
func (c *Client) ListResourceAIProviders(ctx context.Context, resourceID int64) ([]ResourceAIProvider, error) {
	var out listResourceAIProvidersResult
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/resource/%d/ai-providers", resourceID), nil, &out); err != nil {
		return nil, err
	}
	return out.Providers, nil
}
