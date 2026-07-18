package models

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
	UserID string `json:"user_id"`
	Token  string `json:"token"`
}
