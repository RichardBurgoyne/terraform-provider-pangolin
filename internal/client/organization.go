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

	// Settings. Nullable columns come back as JSON null, hence the pointers.
	// A server that predates a retention setting doesn't return it, which
	// reads as zero.
	RequireTwoFactor           *bool  `json:"requireTwoFactor"`
	MaxSessionLengthHours      *int64 `json:"maxSessionLengthHours"`
	PasswordExpiryDays         *int64 `json:"passwordExpiryDays"`
	LogRetentionDaysRequest    int64  `json:"settingsLogRetentionDaysRequest"`
	LogRetentionDaysAccess     int64  `json:"settingsLogRetentionDaysAccess"`
	LogRetentionDaysAction     int64  `json:"settingsLogRetentionDaysAction"`
	LogRetentionDaysConnection int64  `json:"settingsLogRetentionDaysConnection"`
	LogRetentionDaysAISessions int64  `json:"settingsLogRetentionDaysAISessions"`
	LogRetentionDaysSSH        int64  `json:"settingsLogRetentionDaysSsh"`
	EnableGlobalNewtAutoUpdate bool   `json:"settingsEnableGlobalNewtAutoUpdate"`
}

// UpdateOrganizationRequest changes only the fields that are set. The server's
// schema is strict, so a field it doesn't know (a setting newer than the
// server) fails the whole request and must be left nil.
type UpdateOrganizationRequest struct {
	Name                       *string `json:"name,omitempty"`
	RequireTwoFactor           *bool   `json:"requireTwoFactor,omitempty"`
	MaxSessionLengthHours      *int64  `json:"maxSessionLengthHours,omitempty"`
	PasswordExpiryDays         *int64  `json:"passwordExpiryDays,omitempty"`
	LogRetentionDaysRequest    *int64  `json:"settingsLogRetentionDaysRequest,omitempty"`
	LogRetentionDaysAccess     *int64  `json:"settingsLogRetentionDaysAccess,omitempty"`
	LogRetentionDaysAction     *int64  `json:"settingsLogRetentionDaysAction,omitempty"`
	LogRetentionDaysConnection *int64  `json:"settingsLogRetentionDaysConnection,omitempty"`
	LogRetentionDaysAISessions *int64  `json:"settingsLogRetentionDaysAISessions,omitempty"`
	LogRetentionDaysSSH        *int64  `json:"settingsLogRetentionDaysSsh,omitempty"`
	EnableGlobalNewtAutoUpdate *bool   `json:"settingsEnableGlobalNewtAutoUpdate,omitempty"`
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
	if err := c.do(ctx, http.MethodGet, "/org/"+pathEscape(orgID), nil, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Org, nil
}

func (c *Client) UpdateOrganization(ctx context.Context, orgID string, in UpdateOrganizationRequest) error {
	return c.do(ctx, http.MethodPost, "/org/"+pathEscape(orgID), in, nil)
}

func (c *Client) DeleteOrganization(ctx context.Context, orgID string) error {
	return c.do(ctx, http.MethodDelete, "/org/"+pathEscape(orgID), nil, nil)
}
