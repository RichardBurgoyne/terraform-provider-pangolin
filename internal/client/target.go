package client

import (
	"context"
	"encoding/json"
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

	// Health check fields. hcHeaders is a JSON array of {name, value}
	// objects, but the Pangolin API is inconsistent about how it's encoded
	// on the wire: GET /target/:targetId returns it as a real JSON array,
	// while the create/update responses return it as a JSON-encoded string
	// (the raw database column value, un-parsed). It's typed as a raw string
	// here and decoded with DecodeHCHeaders, which handles both shapes.
	HCEnabled            bool   `json:"hcEnabled"`
	HCPath               string `json:"hcPath"`
	HCScheme             string `json:"hcScheme"`
	HCMode               string `json:"hcMode"`
	HCHostname           string `json:"hcHostname"`
	HCPort               int64  `json:"hcPort"`
	HCInterval           int64  `json:"hcInterval"`
	HCUnhealthyInterval  int64  `json:"hcUnhealthyInterval"`
	HCTimeout            int64  `json:"hcTimeout"`
	HCHeaders            any    `json:"hcHeaders"`
	HCFollowRedirects    bool   `json:"hcFollowRedirects"`
	HCMethod             string `json:"hcMethod"`
	HCStatus             int64  `json:"hcStatus"`
	HCHealth             string `json:"hcHealth"` // computed: unknown, healthy, unhealthy
	HCTlsServerName      string `json:"hcTlsServerName"`
	HCHealthyThreshold   int64  `json:"hcHealthyThreshold"`
	HCUnhealthyThreshold int64  `json:"hcUnhealthyThreshold"`
}

// HCHeader is a single header sent with health check requests.
type HCHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// DecodeHCHeaders normalizes Target.HCHeaders (which may come back from the
// API as a JSON array or as a JSON-encoded string containing that array,
// depending on which endpoint returned it) into a []HCHeader slice.
func DecodeHCHeaders(raw any) ([]HCHeader, error) {
	var headers []HCHeader
	switch v := raw.(type) {
	case nil:
		return nil, nil
	case string:
		if v == "" {
			return nil, nil
		}
		if err := json.Unmarshal([]byte(v), &headers); err != nil {
			return nil, err
		}
	default:
		b, err := json.Marshal(raw)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, &headers); err != nil {
			return nil, err
		}
	}
	return headers, nil
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

	HCEnabled            *bool      `json:"hcEnabled,omitempty"`
	HCPath               *string    `json:"hcPath,omitempty"`
	HCScheme             *string    `json:"hcScheme,omitempty"`
	HCMode               *string    `json:"hcMode,omitempty"`
	HCHostname           *string    `json:"hcHostname,omitempty"`
	HCPort               *int64     `json:"hcPort,omitempty"`
	HCInterval           *int64     `json:"hcInterval,omitempty"`
	HCUnhealthyInterval  *int64     `json:"hcUnhealthyInterval,omitempty"`
	HCTimeout            *int64     `json:"hcTimeout,omitempty"`
	HCHeaders            []HCHeader `json:"hcHeaders,omitempty"`
	HCFollowRedirects    *bool      `json:"hcFollowRedirects,omitempty"`
	HCMethod             *string    `json:"hcMethod,omitempty"`
	HCStatus             *int64     `json:"hcStatus,omitempty"`
	HCTlsServerName      *string    `json:"hcTlsServerName,omitempty"`
	HCHealthyThreshold   *int64     `json:"hcHealthyThreshold,omitempty"`
	HCUnhealthyThreshold *int64     `json:"hcUnhealthyThreshold,omitempty"`
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

	HCEnabled            *bool      `json:"hcEnabled,omitempty"`
	HCPath               *string    `json:"hcPath,omitempty"`
	HCScheme             *string    `json:"hcScheme,omitempty"`
	HCMode               *string    `json:"hcMode,omitempty"`
	HCHostname           *string    `json:"hcHostname,omitempty"`
	HCPort               *int64     `json:"hcPort,omitempty"`
	HCInterval           *int64     `json:"hcInterval,omitempty"`
	HCUnhealthyInterval  *int64     `json:"hcUnhealthyInterval,omitempty"`
	HCTimeout            *int64     `json:"hcTimeout,omitempty"`
	HCHeaders            []HCHeader `json:"hcHeaders,omitempty"`
	HCFollowRedirects    *bool      `json:"hcFollowRedirects,omitempty"`
	HCMethod             *string    `json:"hcMethod,omitempty"`
	HCStatus             *int64     `json:"hcStatus,omitempty"`
	HCTlsServerName      *string    `json:"hcTlsServerName,omitempty"`
	HCHealthyThreshold   *int64     `json:"hcHealthyThreshold,omitempty"`
	HCUnhealthyThreshold *int64     `json:"hcUnhealthyThreshold,omitempty"`
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
