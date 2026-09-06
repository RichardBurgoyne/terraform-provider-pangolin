package client

import (
	"context"
	"fmt"
	"net/http"
)

type Site struct {
	SiteID                int64  `json:"siteId"`
	NiceID                string `json:"niceId"`
	Name                  string `json:"name"`
	Type                  string `json:"type"`
	ExitNodeID            *int64 `json:"exitNodeId"`
	PubKey                string `json:"pubKey"`
	Subnet                string `json:"subnet"`
	Address               string `json:"address"`
	DockerSocketEnabled   bool   `json:"dockerSocketEnabled"`
	AutoUpdateEnabled     bool   `json:"autoUpdateEnabled"`
	AutoUpdateOverrideOrg bool   `json:"autoUpdateOverrideOrg"`
	NewtID                string `json:"newtId"`
	Secret                string `json:"secret"`
}

type CreateSiteRequest struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	ExitNodeID *int64 `json:"exitNodeId,omitempty"`
	NiceID     string `json:"niceId,omitempty"`
	PubKey     string `json:"pubKey,omitempty"`
	Subnet     string `json:"subnet,omitempty"`
	Address    string `json:"address,omitempty"`
}

type UpdateSiteRequest struct {
	Name                  *string `json:"name,omitempty"`
	NiceID                *string `json:"niceId,omitempty"`
	DockerSocketEnabled   *bool   `json:"dockerSocketEnabled,omitempty"`
	AutoUpdateEnabled     *bool   `json:"autoUpdateEnabled,omitempty"`
	AutoUpdateOverrideOrg *bool   `json:"autoUpdateOverrideOrg,omitempty"`
}

func (c *Client) CreateSite(ctx context.Context, orgID string, in CreateSiteRequest) (*Site, error) {
	var out Site
	if err := c.do(ctx, http.MethodPut, "/org/"+orgID+"/site", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetSite(ctx context.Context, siteID int64) (*Site, error) {
	var out Site
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/site/%d", siteID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateSite(ctx context.Context, siteID int64, in UpdateSiteRequest) (*Site, error) {
	var out Site
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/site/%d", siteID), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteSite(ctx context.Context, siteID int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/site/%d", siteID), nil, nil)
}
