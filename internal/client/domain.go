package client

import (
	"context"
	"net/http"
)

type Domain struct {
	DomainID           string `json:"domainId"`
	BaseDomain         string `json:"baseDomain"`
	Type               string `json:"type"`
	Verified           bool   `json:"verified"`
	ConfigManaged      bool   `json:"configManaged"`
	CertResolver       string `json:"certResolver"`
	PreferWildcardCert bool   `json:"preferWildcardCert"`
}

type CreateDomainRequest struct {
	Type               string `json:"type"`
	BaseDomain         string `json:"baseDomain"`
	CertResolver       string `json:"certResolver,omitempty"`
	PreferWildcardCert *bool  `json:"preferWildcardCert,omitempty"`
}

type CreateDomainResult struct {
	DomainID string `json:"domainId"`
}

type UpdateDomainRequest struct {
	CertResolver       string `json:"certResolver,omitempty"`
	PreferWildcardCert *bool  `json:"preferWildcardCert,omitempty"`
}

func (c *Client) CreateDomain(ctx context.Context, orgID string, in CreateDomainRequest) (*CreateDomainResult, error) {
	var out CreateDomainResult
	if err := c.do(ctx, http.MethodPut, "/org/"+pathEscape(orgID)+"/domain", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetDomain(ctx context.Context, orgID, domainID string) (*Domain, error) {
	var out Domain
	if err := c.do(ctx, http.MethodGet, "/org/"+pathEscape(orgID)+"/domain/"+pathEscape(domainID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateDomain(ctx context.Context, orgID, domainID string, in UpdateDomainRequest) (*Domain, error) {
	var out Domain
	if err := c.do(ctx, http.MethodPost, "/org/"+pathEscape(orgID)+"/domain/"+pathEscape(domainID), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteDomain(ctx context.Context, orgID, domainID string) error {
	return c.do(ctx, http.MethodDelete, "/org/"+pathEscape(orgID)+"/domain/"+pathEscape(domainID), nil, nil)
}
