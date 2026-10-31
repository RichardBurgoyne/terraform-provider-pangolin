package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

const listPageSize = 200

type Label struct {
	LabelID int64  `json:"labelId"`
	Name    string `json:"name"`
	Color   string `json:"color"`
}

type LabelRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// LabelTarget identifies what a label is attached to. Exactly one field is
// set.
type LabelTarget struct {
	SiteID         *int64 `json:"siteId,omitempty"`
	ResourceID     *int64 `json:"resourceId,omitempty"`
	SiteResourceID *int64 `json:"siteResourceId,omitempty"`
	ClientID       *int64 `json:"clientId,omitempty"`
}

type labelResult struct {
	Label Label `json:"label"`
}

func labelPath(orgID string, labelID int64) string {
	return fmt.Sprintf("/org/%s/label/%d", pathEscape(orgID), labelID)
}

func (c *Client) CreateLabel(ctx context.Context, orgID string, in LabelRequest) (*Label, error) {
	var out labelResult
	if err := c.do(ctx, http.MethodPost, "/org/"+pathEscape(orgID)+"/labels", in, &out); err != nil {
		return nil, err
	}
	return &out.Label, nil
}

// GetLabel returns the label with the given ID, or a 404 APIError if the org
// has no such label. The API has no single-label GET, so this walks the list.
func (c *Client) GetLabel(ctx context.Context, orgID string, labelID int64) (*Label, error) {
	for page := 1; ; page++ {
		var out struct {
			Labels []Label `json:"labels"`
		}
		path := fmt.Sprintf("/org/%s/labels?pageSize=%d&page=%d", pathEscape(orgID), listPageSize, page)
		if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
			return nil, err
		}
		for i := range out.Labels {
			if out.Labels[i].LabelID == labelID {
				return &out.Labels[i], nil
			}
		}
		if len(out.Labels) < listPageSize {
			return nil, &APIError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("label %d not found", labelID)}
		}
	}
}

func (c *Client) UpdateLabel(ctx context.Context, orgID string, labelID int64, in LabelRequest) (*Label, error) {
	var out labelResult
	if err := c.do(ctx, http.MethodPatch, labelPath(orgID, labelID), in, &out); err != nil {
		return nil, err
	}
	return &out.Label, nil
}

func (c *Client) DeleteLabel(ctx context.Context, orgID string, labelID int64) error {
	return c.do(ctx, http.MethodDelete, labelPath(orgID, labelID), nil, nil)
}

func (c *Client) AttachLabel(ctx context.Context, orgID string, labelID int64, target LabelTarget) error {
	return c.do(ctx, http.MethodPut, labelPath(orgID, labelID)+"/attach", target, nil)
}

func (c *Client) DetachLabel(ctx context.Context, orgID string, labelID int64, target LabelTarget) error {
	return c.do(ctx, http.MethodPut, labelPath(orgID, labelID)+"/detach", target, nil)
}

// labeledItem decodes one row of any of the org list endpoints. Only the ID
// that matches the endpoint is populated.
type labeledItem struct {
	SiteID         int64   `json:"siteId"`
	ResourceID     int64   `json:"resourceId"`
	SiteResourceID int64   `json:"siteResourceId"`
	ClientID       int64   `json:"clientId"`
	Labels         []Label `json:"labels"`
}

// IsLabelAttached reports whether the label is attached to target. No endpoint
// answers this directly: labels are only returned on the org list endpoints,
// so this walks the list for the target's type. A missing target reads as not
// attached.
func (c *Client) IsLabelAttached(ctx context.Context, orgID string, labelID int64, target LabelTarget) (bool, error) {
	var endpoint, key string
	var match func(labeledItem) bool
	query := url.Values{"pageSize": {strconv.Itoa(listPageSize)}}
	switch {
	case target.SiteID != nil:
		endpoint, key = "sites", "sites"
		match = func(i labeledItem) bool { return i.SiteID == *target.SiteID }
	case target.ResourceID != nil:
		endpoint, key = "resources", "resources"
		match = func(i labeledItem) bool { return i.ResourceID == *target.ResourceID }
	case target.SiteResourceID != nil:
		endpoint, key = "site-resources", "siteResources"
		match = func(i labeledItem) bool { return i.SiteResourceID == *target.SiteResourceID }
	case target.ClientID != nil:
		endpoint, key = "clients", "clients"
		match = func(i labeledItem) bool { return i.ClientID == *target.ClientID }
		// The clients list only returns active clients by default.
		query.Set("status", "active,blocked,archived")
	default:
		return false, fmt.Errorf("label target has no ID set")
	}

	for page := 1; ; page++ {
		query.Set("page", strconv.Itoa(page))
		var out map[string]json.RawMessage
		if err := c.do(ctx, http.MethodGet, "/org/"+pathEscape(orgID)+"/"+endpoint+"?"+query.Encode(), nil, &out); err != nil {
			return false, err
		}
		var items []labeledItem
		if raw, ok := out[key]; ok {
			if err := json.Unmarshal(raw, &items); err != nil {
				return false, fmt.Errorf("decoding %s: %w", key, err)
			}
		}
		for _, item := range items {
			if !match(item) {
				continue
			}
			for _, label := range item.Labels {
				if label.LabelID == labelID {
					return true, nil
				}
			}
			return false, nil
		}
		if len(items) < listPageSize {
			return false, nil
		}
	}
}
