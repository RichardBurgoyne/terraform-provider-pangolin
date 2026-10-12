package client

import (
	"context"
	"fmt"
	"net/http"
)

// SiteResource is a private resource: a host, CIDR range, HTTP(S) service,
// SSH target or gateway reachable by clients through one or more sites.
// Nullable columns are pointers.
type SiteResource struct {
	SiteResourceID     int64   `json:"siteResourceId"`
	OrgID              string  `json:"orgId"`
	NiceID             string  `json:"niceId"`
	Name               string  `json:"name"`
	Mode               string  `json:"mode"`
	Scheme             *string `json:"scheme"`
	SSL                bool    `json:"ssl"`
	Destination        *string `json:"destination"`
	DestinationPort    *int64  `json:"destinationPort"`
	Enabled            bool    `json:"enabled"`
	Alias              *string `json:"alias"`
	AliasAddress       *string `json:"aliasAddress"`
	TCPPortRangeString string  `json:"tcpPortRangeString"`
	UDPPortRangeString string  `json:"udpPortRangeString"`
	DisableICMP        bool    `json:"disableIcmp"`
	AuthDaemonPort     *int64  `json:"authDaemonPort"`
	AuthDaemonMode     *string `json:"authDaemonMode"`
	PamMode            *string `json:"pamMode"`
	DomainID           *string `json:"domainId"`
	Subdomain          *string `json:"subdomain"`
	FullDomain         *string `json:"fullDomain"`
}

type CreateSiteResourceRequest struct {
	Name               string   `json:"name"`
	NiceID             string   `json:"niceId,omitempty"`
	Mode               string   `json:"mode"`
	SiteIDs            []int64  `json:"siteIds,omitempty"`
	Destination        *string  `json:"destination,omitempty"`
	DestinationPort    *int64   `json:"destinationPort,omitempty"`
	Scheme             *string  `json:"scheme,omitempty"`
	SSL                *bool    `json:"ssl,omitempty"`
	Alias              *string  `json:"alias,omitempty"`
	TCPPortRangeString *string  `json:"tcpPortRangeString,omitempty"`
	UDPPortRangeString *string  `json:"udpPortRangeString,omitempty"`
	DisableICMP        *bool    `json:"disableIcmp,omitempty"`
	AuthDaemonPort     *int64   `json:"authDaemonPort,omitempty"`
	AuthDaemonMode     *string  `json:"authDaemonMode,omitempty"`
	PamMode            *string  `json:"pamMode,omitempty"`
	DomainID           *string  `json:"domainId,omitempty"`
	Subdomain          *string  `json:"subdomain,omitempty"`
	UserIDs            []string `json:"userIds"`
	RoleIDs            []int64  `json:"roleIds"`
	ClientIDs          []int64  `json:"clientIds"`
}

// UpdateSiteResourceRequest is a partial update: nil fields are left alone.
// UserIDs, RoleIDs and ClientIDs replace the full assignment when non-nil.
type UpdateSiteResourceRequest struct {
	Name               *string   `json:"name,omitempty"`
	Mode               *string   `json:"mode,omitempty"`
	SiteIDs            []int64   `json:"siteIds,omitempty"`
	Destination        *string   `json:"destination,omitempty"`
	DestinationPort    *int64    `json:"destinationPort,omitempty"`
	Scheme             *string   `json:"scheme,omitempty"`
	SSL                *bool     `json:"ssl,omitempty"`
	Enabled            *bool     `json:"enabled,omitempty"`
	Alias              *string   `json:"alias,omitempty"`
	TCPPortRangeString *string   `json:"tcpPortRangeString,omitempty"`
	UDPPortRangeString *string   `json:"udpPortRangeString,omitempty"`
	DisableICMP        *bool     `json:"disableIcmp,omitempty"`
	AuthDaemonPort     *int64    `json:"authDaemonPort,omitempty"`
	AuthDaemonMode     *string   `json:"authDaemonMode,omitempty"`
	PamMode            *string   `json:"pamMode,omitempty"`
	DomainID           *string   `json:"domainId,omitempty"`
	Subdomain          *string   `json:"subdomain,omitempty"`
	UserIDs            *[]string `json:"userIds,omitempty"`
	RoleIDs            *[]int64  `json:"roleIds,omitempty"`
	ClientIDs          *[]int64  `json:"clientIds,omitempty"`
}

func (c *Client) CreateSiteResource(ctx context.Context, orgID string, in CreateSiteResourceRequest) (*SiteResource, error) {
	if in.UserIDs == nil {
		in.UserIDs = []string{}
	}
	if in.RoleIDs == nil {
		in.RoleIDs = []int64{}
	}
	if in.ClientIDs == nil {
		in.ClientIDs = []int64{}
	}
	var out SiteResource
	if err := c.do(ctx, http.MethodPut, "/org/"+pathEscape(orgID)+"/site-resource", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetSiteResource(ctx context.Context, id int64) (*SiteResource, error) {
	var out SiteResource
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/site-resource/%d", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateSiteResource(ctx context.Context, id int64, in UpdateSiteResourceRequest) (*SiteResource, error) {
	var out SiteResource
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/site-resource/%d", id), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteSiteResource(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/site-resource/%d", id), nil, nil)
}

// ListSiteResourceRoleIDs returns the non-admin roles assigned to a site
// resource. Pangolin always keeps the admin role attached, so it is excluded
// to match what a caller configures.
func (c *Client) ListSiteResourceRoleIDs(ctx context.Context, id int64) ([]int64, error) {
	var out struct {
		Roles []struct {
			RoleID  int64 `json:"roleId"`
			IsAdmin bool  `json:"isAdmin"`
		} `json:"roles"`
	}
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/site-resource/%d/roles", id), nil, &out); err != nil {
		return nil, err
	}
	ids := []int64{}
	for _, r := range out.Roles {
		if !r.IsAdmin {
			ids = append(ids, r.RoleID)
		}
	}
	return ids, nil
}

func (c *Client) ListSiteResourceUserIDs(ctx context.Context, id int64) ([]string, error) {
	var out struct {
		Users []struct {
			UserID string `json:"userId"`
		} `json:"users"`
	}
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/site-resource/%d/users", id), nil, &out); err != nil {
		return nil, err
	}
	ids := []string{}
	for _, u := range out.Users {
		ids = append(ids, u.UserID)
	}
	return ids, nil
}

func (c *Client) ListSiteResourceClientIDs(ctx context.Context, id int64) ([]int64, error) {
	var out struct {
		Clients []struct {
			ClientID int64 `json:"clientId"`
		} `json:"clients"`
	}
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/site-resource/%d/clients", id), nil, &out); err != nil {
		return nil, err
	}
	ids := []int64{}
	for _, cl := range out.Clients {
		ids = append(ids, cl.ClientID)
	}
	return ids, nil
}

// ListSiteResourceSiteIDs returns the sites a site resource is served
// through. The single-resource GET doesn't include them, so this pages
// through the org-wide listing.
func (c *Client) ListSiteResourceSiteIDs(ctx context.Context, orgID string, id int64) ([]int64, error) {
	const pageSize = 100
	for page := 1; ; page++ {
		var out struct {
			SiteResources []struct {
				SiteResourceID int64   `json:"siteResourceId"`
				SiteIDs        []int64 `json:"siteIds"`
			} `json:"siteResources"`
			Pagination struct {
				Total int `json:"total"`
			} `json:"pagination"`
		}
		path := fmt.Sprintf("/org/%s/site-resources?pageSize=%d&page=%d", pathEscape(orgID), pageSize, page)
		if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
			return nil, err
		}
		for _, sr := range out.SiteResources {
			if sr.SiteResourceID == id {
				if sr.SiteIDs == nil {
					return []int64{}, nil
				}
				return sr.SiteIDs, nil
			}
		}
		if page*pageSize >= out.Pagination.Total || len(out.SiteResources) == 0 {
			return nil, &APIError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("site resource %d not found in org %s", id, orgID)}
		}
	}
}
