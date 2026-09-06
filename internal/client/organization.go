package client

import (
	"context"
	"net/http"
)

type Organization struct {
	OrgID         string `json:"orgId"`
	Name          string `json:"name"`
	Subnet        string `json:"subnet"`
	UtilitySubnet string `json:"utilitySubnet"`
}

type CreateOrganizationRequest struct {
	OrgID         string `json:"orgId"`
	Name          string `json:"name"`
	Subnet        string `json:"subnet"`
	UtilitySubnet string `json:"utilitySubnet"`
}

func (c *Client) CreateOrganization(ctx context.Context, in CreateOrganizationRequest) (*Organization, error) {
	var out Organization
	if err := c.do(ctx, http.MethodPut, "/org", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetOrganization(ctx context.Context, orgID string) (*Organization, error) {
	var wrapper struct {
		Org Organization `json:"org"`
	}
	if err := c.do(ctx, http.MethodGet, "/org/"+orgID, nil, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Org, nil
}

func (c *Client) DeleteOrganization(ctx context.Context, orgID string) error {
	return c.do(ctx, http.MethodDelete, "/org/"+orgID, nil, nil)
}
