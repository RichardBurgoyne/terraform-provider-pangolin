package client

import (
	"context"
	"fmt"
	"net/http"
)

type User struct {
	UserID          string `json:"userId"`
	Username        string `json:"username"`
	Email           string `json:"email"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	IdpID           int64  `json:"idpId"`
	AutoProvisioned bool   `json:"autoProvisioned"`
}

type CreateOrgUserRequest struct {
	Username string  `json:"username"`
	Email    string  `json:"email,omitempty"`
	Name     string  `json:"name,omitempty"`
	Type     string  `json:"type"`
	IdpID    int64   `json:"idpId"`
	RoleIDs  []int64 `json:"roleIds"`
}

type UpdateOrgUserRequest struct {
	AutoProvisioned *bool `json:"autoProvisioned,omitempty"`
}

type listUsersResult struct {
	Users []User `json:"users"`
}

func (c *Client) CreateOrgUser(ctx context.Context, orgID string, in CreateOrgUserRequest) error {
	return c.do(ctx, http.MethodPut, "/org/"+orgID+"/user", in, nil)
}

func (c *Client) ListUsers(ctx context.Context, orgID string) ([]User, error) {
	var out listUsersResult
	if err := c.do(ctx, http.MethodGet, "/org/"+orgID+"/users", nil, &out); err != nil {
		return nil, err
	}
	return out.Users, nil
}

func (c *Client) GetUserByUsername(ctx context.Context, orgID, username string) (*User, error) {
	users, err := c.ListUsers(ctx, orgID)
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		if u.Username == username {
			return &u, nil
		}
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("user %q not found in org %s", username, orgID)}
}

func (c *Client) GetOrgUser(ctx context.Context, orgID, userID string) (*User, error) {
	var out User
	if err := c.do(ctx, http.MethodGet, "/org/"+orgID+"/user/"+userID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateOrgUser(ctx context.Context, orgID, userID string, in UpdateOrgUserRequest) error {
	return c.do(ctx, http.MethodPost, "/org/"+orgID+"/user/"+userID, in, nil)
}

// AddUserRole adds one role to a user.
func (c *Client) AddUserRole(ctx context.Context, roleID int64, userID string) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/role/%d/add/%s", roleID, userID), nil, nil)
}

// RemoveUserRole removes one role from a user. This route
// (DELETE /user/:userId/remove-role/:roleId) is part of Pangolin's
// commercial integration API, not the AGPL-licensed community build; against
// a plain self-hosted community instance it returns 404, not a partial
// success.
func (c *Client) RemoveUserRole(ctx context.Context, userID string, roleID int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/user/%s/remove-role/%d", userID, roleID), nil, nil)
}

func (c *Client) DeleteOrgUser(ctx context.Context, orgID, userID string) error {
	return c.do(ctx, http.MethodDelete, "/org/"+orgID+"/user/"+userID, nil, nil)
}
