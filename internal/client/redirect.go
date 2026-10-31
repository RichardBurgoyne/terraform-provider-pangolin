package client

import (
	"context"
	"fmt"
	"net/http"
)

// Redirect sends requests for a resource's domain, or for a domain of its
// own, to another host. Exactly one of ResourceID and DomainID is set.
// Nullable columns come back as JSON null, hence the pointers.
type Redirect struct {
	RedirectID      int64   `json:"redirectId"`
	OrgID           string  `json:"orgId"`
	NiceID          string  `json:"niceId"`
	Name            string  `json:"name"`
	ResourceID      *int64  `json:"resourceId"`
	DomainID        *string `json:"domainId"`
	Subdomain       *string `json:"subdomain"`
	DestinationHost string  `json:"destinationHost"`
	PathMatchType   string  `json:"pathMatchType"`
	MatchPath       *string `json:"matchPath"`
	RewritePath     *string `json:"rewritePath"`
	RewritePathType *string `json:"rewritePathType"`
	Priority        *int64  `json:"priority"`
	Permanent       bool    `json:"permanent"`
	SSL             bool    `json:"ssl"`
	Enabled         bool    `json:"enabled"`
}

// RedirectRequest is the body for both create and update. The nullable fields
// have no omitempty because an update clears them by sending an explicit null.
type RedirectRequest struct {
	Name            string  `json:"name"`
	ResourceID      *int64  `json:"resourceId"`
	DomainID        *string `json:"domainId"`
	Subdomain       *string `json:"subdomain"`
	DestinationHost string  `json:"destinationHost"`
	PathMatchType   *string `json:"pathMatchType,omitempty"`
	MatchPath       *string `json:"matchPath"`
	RewritePath     *string `json:"rewritePath"`
	RewritePathType *string `json:"rewritePathType"`
	Priority        *int64  `json:"priority,omitempty"`
	Permanent       *bool   `json:"permanent,omitempty"`
	SSL             *bool   `json:"ssl,omitempty"`
	Enabled         *bool   `json:"enabled,omitempty"`
}

type redirectResult struct {
	Redirect Redirect `json:"redirect"`
}

func redirectPath(orgID string, redirectID int64) string {
	return fmt.Sprintf("/org/%s/redirects/%d", pathEscape(orgID), redirectID)
}

func (c *Client) CreateRedirect(ctx context.Context, orgID string, in RedirectRequest) (*Redirect, error) {
	var out redirectResult
	if err := c.do(ctx, http.MethodPut, "/org/"+pathEscape(orgID)+"/redirect", in, &out); err != nil {
		return nil, err
	}
	return &out.Redirect, nil
}

func (c *Client) GetRedirect(ctx context.Context, orgID string, redirectID int64) (*Redirect, error) {
	var out redirectResult
	if err := c.do(ctx, http.MethodGet, redirectPath(orgID, redirectID), nil, &out); err != nil {
		return nil, err
	}
	return &out.Redirect, nil
}

func (c *Client) UpdateRedirect(ctx context.Context, orgID string, redirectID int64, in RedirectRequest) (*Redirect, error) {
	var out redirectResult
	if err := c.do(ctx, http.MethodPost, redirectPath(orgID, redirectID), in, &out); err != nil {
		return nil, err
	}
	return &out.Redirect, nil
}

func (c *Client) DeleteRedirect(ctx context.Context, orgID string, redirectID int64) error {
	return c.do(ctx, http.MethodDelete, redirectPath(orgID, redirectID), nil, nil)
}
