package client

import (
	"context"
	"fmt"
	"net/http"
)

type AIProvider struct {
	ProviderID           int64      `json:"providerId"`
	OrgID                string     `json:"orgId"`
	Name                 string     `json:"name"`
	NiceID               string     `json:"niceId"`
	Type                 string     `json:"type"`
	UpstreamURL          string     `json:"upstreamUrl"`
	APIKey               string     `json:"apiKey"`
	AuthType             string     `json:"authType"`
	RoutingMode          string     `json:"routingMode"`
	Capabilities         []string   `json:"capabilities"`
	Headers              []HCHeader `json:"headers"`
	SkipTLSVerification  bool       `json:"skipTlsVerification"`
	Enabled              bool       `json:"enabled"`
	EffectiveUpstreamURL string     `json:"effectiveUpstreamUrl"`
	EffectiveAuthType    string     `json:"effectiveAuthType"`
}

type CreateAIProviderRequest struct {
	Name                string     `json:"name"`
	Type                string     `json:"type"`
	UpstreamURL         *string    `json:"upstreamUrl,omitempty"`
	APIKey              *string    `json:"apiKey,omitempty"`
	AuthType            string     `json:"authType,omitempty"`
	RoutingMode         string     `json:"routingMode,omitempty"`
	Capabilities        []string   `json:"capabilities,omitempty"`
	Headers             []HCHeader `json:"headers,omitempty"`
	SkipTLSVerification *bool      `json:"skipTlsVerification,omitempty"`
	Enabled             *bool      `json:"enabled,omitempty"`
}

type UpdateAIProviderRequest struct {
	Name                *string    `json:"name,omitempty"`
	NiceID              *string    `json:"niceId,omitempty"`
	UpstreamURL         *string    `json:"upstreamUrl,omitempty"`
	APIKey              *string    `json:"apiKey,omitempty"`
	AuthType            string     `json:"authType,omitempty"`
	RoutingMode         string     `json:"routingMode,omitempty"`
	Capabilities        []string   `json:"capabilities,omitempty"`
	Headers             []HCHeader `json:"headers,omitempty"`
	SkipTLSVerification *bool      `json:"skipTlsVerification,omitempty"`
	Enabled             *bool      `json:"enabled,omitempty"`
}

type createOrGetAIProviderResult struct {
	Provider AIProvider `json:"provider"`
}

func (c *Client) CreateAIProvider(ctx context.Context, orgID string, in CreateAIProviderRequest) (*AIProvider, error) {
	var out createOrGetAIProviderResult
	if err := c.do(ctx, http.MethodPut, "/org/"+pathEscape(orgID)+"/ai-provider", in, &out); err != nil {
		return nil, err
	}
	return &out.Provider, nil
}

func (c *Client) GetAIProvider(ctx context.Context, providerID int64) (*AIProvider, error) {
	var out createOrGetAIProviderResult
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/ai-provider/%d", providerID), nil, &out); err != nil {
		return nil, err
	}
	return &out.Provider, nil
}

func (c *Client) UpdateAIProvider(ctx context.Context, providerID int64, in UpdateAIProviderRequest) (*AIProvider, error) {
	var out createOrGetAIProviderResult
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/ai-provider/%d", providerID), in, &out); err != nil {
		return nil, err
	}
	return &out.Provider, nil
}

func (c *Client) DeleteAIProvider(ctx context.Context, providerID int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/ai-provider/%d", providerID), nil, nil)
}
