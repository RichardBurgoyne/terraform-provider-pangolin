package client

import (
	"context"
	"fmt"
	"net/http"
)

type PangolinResource struct {
	ResourceID     int64  `json:"resourceId"`
	NiceID         string `json:"niceId"`
	Name           string `json:"name"`
	Mode           string `json:"mode"`
	DomainID       string `json:"domainId"`
	FullDomain     string `json:"fullDomain"`
	Subdomain      string `json:"subdomain"`
	StickySession  bool   `json:"stickySession"`
	PostAuthPath   string `json:"postAuthPath"`
	PamMode        string `json:"pamMode"`
	AuthDaemonMode string `json:"authDaemonMode"`
	AuthDaemonPort int64  `json:"authDaemonPort"`
	Enabled        bool   `json:"enabled"`
	SSL            bool   `json:"ssl"`
}

type CreateResourceRequest struct {
	Name           string `json:"name"`
	Mode           string `json:"mode"`
	DomainID       string `json:"domainId"`
	Subdomain      string `json:"subdomain,omitempty"`
	StickySession  *bool  `json:"stickySession,omitempty"`
	PostAuthPath   string `json:"postAuthPath,omitempty"`
	PamMode        string `json:"pamMode,omitempty"`
	AuthDaemonMode string `json:"authDaemonMode,omitempty"`
	AuthDaemonPort *int64 `json:"authDaemonPort,omitempty"`
}

type UpdateResourceRequest struct {
	Name           *string `json:"name,omitempty"`
	NiceID         *string `json:"niceId,omitempty"`
	Subdomain      *string `json:"subdomain,omitempty"`
	SSL            *bool   `json:"ssl,omitempty"`
	DomainID       *string `json:"domainId,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
	StickySession  *bool   `json:"stickySession,omitempty"`
	PostAuthPath   *string `json:"postAuthPath,omitempty"`
	PamMode        *string `json:"pamMode,omitempty"`
	AuthDaemonMode *string `json:"authDaemonMode,omitempty"`
	AuthDaemonPort *int64  `json:"authDaemonPort,omitempty"`
}

func (c *Client) CreateResource(ctx context.Context, orgID string, in CreateResourceRequest) (*PangolinResource, error) {
	var out PangolinResource
	if err := c.do(ctx, http.MethodPut, "/org/"+orgID+"/resource", in, &out); err != nil {
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
