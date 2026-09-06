package client

import (
	"context"
	"fmt"
	"net/http"
)

type roleIDBody struct {
	RoleID int64 `json:"roleId"`
}

type listResourceRolesResult struct {
	Roles []struct {
		RoleID int64 `json:"roleId"`
	} `json:"roles"`
}

func (c *Client) AddRoleToResource(ctx context.Context, resourceID, roleID int64) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/resource/%d/roles/add", resourceID), roleIDBody{RoleID: roleID}, nil)
}

func (c *Client) RemoveRoleFromResource(ctx context.Context, resourceID, roleID int64) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/resource/%d/roles/remove", resourceID), roleIDBody{RoleID: roleID}, nil)
}

func (c *Client) ListResourceRoles(ctx context.Context, resourceID int64) ([]int64, error) {
	var out listResourceRolesResult
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/resource/%d/roles", resourceID), nil, &out); err != nil {
		return nil, err
	}
	ids := make([]int64, len(out.Roles))
	for i, role := range out.Roles {
		ids[i] = role.RoleID
	}
	return ids, nil
}
