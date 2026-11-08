package client

import (
	"context"
	"fmt"
	"net/http"
)

// CustomDnsRecord is a DNS record that Pangolin's nameservers serve for a
// delegated domain, alongside the records Pangolin manages itself. Each entry
// in Values is served as a separate resource record. Priority is only set for
// MX and SRV records, Weight and Port only for SRV.
type CustomDnsRecord struct {
	CustomDnsRecordID int64    `json:"customDnsRecordId"`
	DomainID          string   `json:"domainId"`
	Name              string   `json:"name"`
	FullDomain        string   `json:"fullDomain"`
	Type              string   `json:"type"`
	Values            []string `json:"values"`
	TTL               int64    `json:"ttl"`
	Priority          *int64   `json:"priority"`
	Weight            *int64   `json:"weight"`
	Port              *int64   `json:"port"`
	Enabled           bool     `json:"enabled"`
}

// CustomDnsRecordRequest is the body for both create and update. The nullable
// fields have no omitempty because an update clears them by sending an
// explicit null.
type CustomDnsRecordRequest struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Values   []string `json:"values"`
	TTL      *int64   `json:"ttl,omitempty"`
	Priority *int64   `json:"priority"`
	Weight   *int64   `json:"weight"`
	Port     *int64   `json:"port"`
	Enabled  *bool    `json:"enabled,omitempty"`
}

func customDnsRecordBase(orgID, domainID string) string {
	return fmt.Sprintf("/org/%s/domain/%s", pathEscape(orgID), pathEscape(domainID))
}

func (c *Client) CreateCustomDnsRecord(ctx context.Context, orgID, domainID string, in CustomDnsRecordRequest) (*CustomDnsRecord, error) {
	var out CustomDnsRecord
	if err := c.do(ctx, http.MethodPut, customDnsRecordBase(orgID, domainID)+"/custom-dns-record", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListCustomDnsRecords(ctx context.Context, orgID, domainID string) ([]CustomDnsRecord, error) {
	var out struct {
		Records []CustomDnsRecord `json:"records"`
	}
	if err := c.do(ctx, http.MethodGet, customDnsRecordBase(orgID, domainID)+"/custom-dns-records", nil, &out); err != nil {
		return nil, err
	}
	return out.Records, nil
}

// GetCustomDnsRecord finds one record in the domain's list, since the API has
// no endpoint for a single record. A record that isn't listed is reported as
// a 404 APIError so callers can treat it like any other missing object.
func (c *Client) GetCustomDnsRecord(ctx context.Context, orgID, domainID string, recordID int64) (*CustomDnsRecord, error) {
	records, err := c.ListCustomDnsRecords(ctx, orgID, domainID)
	if err != nil {
		return nil, err
	}
	for i := range records {
		if records[i].CustomDnsRecordID == recordID {
			return &records[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("custom DNS record %d not found", recordID)}
}

func (c *Client) UpdateCustomDnsRecord(ctx context.Context, orgID, domainID string, recordID int64, in CustomDnsRecordRequest) (*CustomDnsRecord, error) {
	var out CustomDnsRecord
	path := fmt.Sprintf("%s/custom-dns-record/%d", customDnsRecordBase(orgID, domainID), recordID)
	if err := c.do(ctx, http.MethodPost, path, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteCustomDnsRecord(ctx context.Context, orgID, domainID string, recordID int64) error {
	path := fmt.Sprintf("%s/custom-dns-record/%d", customDnsRecordBase(orgID, domainID), recordID)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}
