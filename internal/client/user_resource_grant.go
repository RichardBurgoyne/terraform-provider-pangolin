package client

import (
	"context"
	"fmt"
	"net/http"
)

type userIDBody struct {
	UserID string `json:"userId"`
}

type listResourceUsersResult struct {
	Users []struct {
		UserID string `json:"userId"`
	} `json:"users"`
}

func (c *Client) AddUserToResource(ctx context.Context, resourceID int64, userID string) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/resource/%d/users/add", resourceID), userIDBody{UserID: userID}, nil)
}

func (c *Client) RemoveUserFromResource(ctx context.Context, resourceID int64, userID string) error {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/resource/%d/users/remove", resourceID), userIDBody{UserID: userID}, nil)
}

func (c *Client) ListResourceUsers(ctx context.Context, resourceID int64) ([]string, error) {
	var out listResourceUsersResult
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/resource/%d/users", resourceID), nil, &out); err != nil {
		return nil, err
	}
	ids := make([]string, len(out.Users))
	for i, u := range out.Users {
		ids[i] = u.UserID
	}
	return ids, nil
}
