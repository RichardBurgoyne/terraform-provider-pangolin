package client

import (
	"context"
	"fmt"
	"net/http"
)

type Target struct {
	TargetID        int64  `json:"targetId"`
	ResourceID      int64  `json:"resourceId"`
	SiteID          int64  `json:"siteId"`
	IP              string `json:"ip"`
	Mode            string `json:"mode"`
	Method          string `json:"method"`
	Port            int64  `json:"port"`
	Enabled         bool   `json:"enabled"`
	Path            string `json:"path"`
	PathMatchType   string `json:"pathMatchType"`
	RewritePath     string `json:"rewritePath"`
	RewritePathType string `json:"rewritePathType"`
	Priority        int64  `json:"priority"`
}

type CreateTargetRequest struct {
	SiteID          int64   `json:"siteId"`
	IP              string  `json:"ip"`
	Mode            string  `json:"mode,omitempty"`
	Method          *string `json:"method,omitempty"`
	Port            int64   `json:"port"`
	Enabled         *bool   `json:"enabled,omitempty"`
	Path            *string `json:"path,omitempty"`
	PathMatchType   *string `json:"pathMatchType,omitempty"`
	RewritePath     *string `json:"rewritePath,omitempty"`
	RewritePathType *string `json:"rewritePathType,omitempty"`
	Priority        *int64  `json:"priority,omitempty"`
}

type UpdateTargetRequest struct {
	SiteID          int64   `json:"siteId"`
	IP              string  `json:"ip"`
	Mode            *string `json:"mode,omitempty"`
	Method          *string `json:"method,omitempty"`
	Port            *int64  `json:"port,omitempty"`
	Enabled         *bool   `json:"enabled,omitempty"`
	Path            *string `json:"path,omitempty"`
	PathMatchType   *string `json:"pathMatchType,omitempty"`
	RewritePath     *string `json:"rewritePath,omitempty"`
	RewritePathType *string `json:"rewritePathType,omitempty"`
	Priority        *int64  `json:"priority,omitempty"`
}

func (c *Client) CreateTarget(ctx context.Context, resourceID int64, in CreateTargetRequest) (*Target, error) {
	var out Target
	if err := c.do(ctx, http.MethodPut, fmt.Sprintf("/resource/%d/target", resourceID), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetTarget(ctx context.Context, targetID int64) (*Target, error) {
	var out Target
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/target/%d", targetID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateTarget(ctx context.Context, targetID int64, in UpdateTargetRequest) (*Target, error) {
	var out Target
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/target/%d", targetID), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteTarget(ctx context.Context, targetID int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/target/%d", targetID), nil, nil)
}
