package client

import (
	"context"
	"fmt"
	"net/http"
)

type Role struct {
	RoleID                int64  `json:"roleId"`
	OrgID                 string `json:"orgId"`
	Name                  string `json:"name"`
	Description           string `json:"description"`
	IsAdmin               bool   `json:"isAdmin"`
	RequireDeviceApproval bool   `json:"requireDeviceApproval"`
	SSHSudoMode           string `json:"sshSudoMode"`
	AllowSSH              bool   `json:"allowSsh"`
}

type CreateRoleRequest struct {
	Name                  string `json:"name"`
	Description           string `json:"description,omitempty"`
	RequireDeviceApproval *bool  `json:"requireDeviceApproval,omitempty"`
	AllowSSH              *bool  `json:"allowSsh,omitempty"`
	SSHSudoMode           string `json:"sshSudoMode,omitempty"`
}

type UpdateRoleRequest struct {
	Name                  *string `json:"name,omitempty"`
	Description           *string `json:"description,omitempty"`
	RequireDeviceApproval *bool   `json:"requireDeviceApproval,omitempty"`
	AllowSSH              *bool   `json:"allowSsh,omitempty"`
	SSHSudoMode           *string `json:"sshSudoMode,omitempty"`
}

type listRolesResult struct {
	Roles []Role `json:"roles"`
}

func (c *Client) CreateRole(ctx context.Context, orgID string, in CreateRoleRequest) (*Role, error) {
	var out Role
	if err := c.do(ctx, http.MethodPut, "/org/"+orgID+"/role", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListRoles(ctx context.Context, orgID string) ([]Role, error) {
	var out listRolesResult
	if err := c.do(ctx, http.MethodGet, "/org/"+orgID+"/roles", nil, &out); err != nil {
		return nil, err
	}
	return out.Roles, nil
}

// GetRoleByID has no dedicated API endpoint (GET /role/:roleId is disabled
// server-side), so it lists every role in the org and filters client-side.
func (c *Client) GetRoleByID(ctx context.Context, orgID string, roleID int64) (*Role, error) {
	roles, err := c.ListRoles(ctx, orgID)
	if err != nil {
		return nil, err
	}
	for _, role := range roles {
		if role.RoleID == roleID {
			return &role, nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("role %d not found in org %s", roleID, orgID)}
}

func (c *Client) UpdateRole(ctx context.Context, roleID int64, in UpdateRoleRequest) (*Role, error) {
	var out Role
	if err := c.do(ctx, http.MethodPost, fmt.Sprintf("/role/%d", roleID), in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteRole(ctx context.Context, roleID int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/role/%d", roleID), nil, nil)
}
