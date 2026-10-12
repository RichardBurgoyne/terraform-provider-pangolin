package client

import (
	"context"
	"fmt"
	"net/http"
)

// ResourceRule is an access rule attached to a resource. Match is one of
// CIDR, IP, PATH, COUNTRY, COUNTRY_IS_NOT, ASN, REGION or METHOD (METHOD was
// added in Pangolin 1.24 and takes a comma-separated list of HTTP methods).
type ResourceRule struct {
	RuleID     int64  `json:"ruleId"`
	ResourceID int64  `json:"resourceId"`
	Action     string `json:"action"`
	Match      string `json:"match"`
	Value      string `json:"value"`
	Priority   int64  `json:"priority"`
	Enabled    bool   `json:"enabled"`
}

type CreateResourceRuleRequest struct {
	Action   string `json:"action"`
	Match    string `json:"match"`
	Value    string `json:"value"`
	Priority int64  `json:"priority"`
	Enabled  *bool  `json:"enabled,omitempty"`
}

type UpdateResourceRuleRequest struct {
	Action   string `json:"action"`
	Match    string `json:"match"`
	Value    string `json:"value"`
	Priority int64  `json:"priority"`
	Enabled  *bool  `json:"enabled,omitempty"`
}

func (c *Client) CreateResourceRule(ctx context.Context, resourceID int64, in CreateResourceRuleRequest) (*ResourceRule, error) {
	var out ResourceRule
	if err := c.do(ctx, http.MethodPut, fmt.Sprintf("/resource/%d/rule", resourceID), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListResourceRules returns every rule on a resource. The API has no
// single-rule GET, so GetResourceRule filters this list.
func (c *Client) ListResourceRules(ctx context.Context, resourceID int64) ([]ResourceRule, error) {
	var out struct {
		Rules []ResourceRule `json:"rules"`
	}
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/resource/%d/rules", resourceID), nil, &out); err != nil {
		return nil, err
	}
	return out.Rules, nil
}

// GetResourceRule returns the rule with the given ID, or a 404 APIError if
// the resource has no such rule.
func (c *Client) GetResourceRule(ctx context.Context, resourceID, ruleID int64) (*ResourceRule, error) {
	rules, err := c.ListResourceRules(ctx, resourceID)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		if rules[i].RuleID == ruleID {
			return &rules[i], nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("rule %d not found on resource %d", ruleID, resourceID)}
}

func (c *Client) UpdateResourceRule(ctx context.Context, resourceID, ruleID int64, in UpdateResourceRuleRequest) (*ResourceRule, error) {
	var out ResourceRule
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/resource/%d/rule/%d", resourceID, ruleID), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteResourceRule(ctx context.Context, resourceID, ruleID int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/resource/%d/rule/%d", resourceID, ruleID), nil, nil)
}
