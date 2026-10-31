package client

import (
	"context"
	"fmt"
	"net/http"
)

// ListClientRoleIDs returns the roles assigned directly to a machine client.
func (c *Client) ListClientRoleIDs(ctx context.Context, clientID int64) ([]int64, error) {
	var out struct {
		Roles []struct {
			RoleID int64 `json:"roleId"`
		} `json:"roles"`
	}
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/client/%d", clientID), nil, &out); err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(out.Roles))
	for _, role := range out.Roles {
		ids = append(ids, role.RoleID)
	}
	return ids, nil
}

// AddClientRole assigns a role to a machine client. This route is part of
// Pangolin's commercial integration API.
func (c *Client) AddClientRole(ctx context.Context, clientID, roleID int64) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/client/%d/add-role/%d", clientID, roleID), nil, nil)
}

// RemoveClientRole removes a role from a machine client. This route is part of
// Pangolin's commercial integration API.
func (c *Client) RemoveClientRole(ctx context.Context, clientID, roleID int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/client/%d/remove-role/%d", clientID, roleID), nil, nil)
}
