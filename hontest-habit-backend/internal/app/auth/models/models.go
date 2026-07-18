package models

import "github.com/insanelyharsh/hontest-habit/internal/types"

type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse is returned by both Signup and Login: either operation
// authenticates the caller immediately.
type AuthResponse struct {
	UserID types.UserId `json:"user_id"`
	Token  string       `json:"token"`
}
