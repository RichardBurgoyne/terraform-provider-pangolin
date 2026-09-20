package client

import (
	"context"
	"net/http"
)

type APIKey struct {
	APIKeyID  string `json:"apiKeyId"`
	Name      string `json:"name"`
	APIKey    string `json:"apiKey"`
	LastChars string `json:"lastChars"`
	CreatedAt string `json:"createdAt"`
}

type CreateAPIKeyRequest struct {
	Name string `json:"name"`
}

type setAPIKeyActionsRequest struct {
	ActionIDs []string `json:"actionIds"`
}

type apiKeyActionsResult struct {
	ActionIDs []string `json:"actionIds"`
}

func (c *Client) CreateAPIKey(ctx context.Context, orgID string, in CreateAPIKeyRequest) (*APIKey, error) {
	var out APIKey
	if err := c.do(ctx, http.MethodPut, "/org/"+pathEscape(orgID)+"/api-key", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetAPIKey(ctx context.Context, orgID, apiKeyID string) (*APIKey, error) {
	var out APIKey
	if err := c.do(ctx, http.MethodGet, "/org/"+pathEscape(orgID)+"/api-key/"+pathEscape(apiKeyID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) SetAPIKeyActions(ctx context.Context, orgID, apiKeyID string, actionIDs []string) error {
	return c.do(ctx, http.MethodPost, "/org/"+pathEscape(orgID)+"/api-key/"+pathEscape(apiKeyID)+"/actions", setAPIKeyActionsRequest{ActionIDs: actionIDs}, nil)
}

func (c *Client) ListAPIKeyActions(ctx context.Context, orgID, apiKeyID string) ([]string, error) {
	var out apiKeyActionsResult
	if err := c.do(ctx, http.MethodGet, "/org/"+pathEscape(orgID)+"/api-key/"+pathEscape(apiKeyID)+"/actions", nil, &out); err != nil {
		return nil, err
	}
	return out.ActionIDs, nil
}

func (c *Client) DeleteAPIKey(ctx context.Context, orgID, apiKeyID string) error {
	return c.do(ctx, http.MethodDelete, "/org/"+pathEscape(orgID)+"/api-key/"+pathEscape(apiKeyID), nil, nil)
}
