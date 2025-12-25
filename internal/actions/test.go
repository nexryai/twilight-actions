package actions

import (
	"errors"
	"fmt"
)

type GetUserRequest struct {
	ID int `json:"id"`
}

type GetUserResponse struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// @action
func GetUser(req GetUserRequest) (GetUserResponse, error) {
	if req.ID == 0 {
		return GetUserResponse{}, errors.New("invalid user id")
	}

	return GetUserResponse{
		Name:  fmt.Sprintf("User-%d", req.ID),
		Email: "user@example.com",
	}, nil
}