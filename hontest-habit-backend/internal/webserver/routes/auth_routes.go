package routes

import (
	"net/http"

	"github.com/insanelyharsh/hontest-habit/internal/app/auth"
	"github.com/insanelyharsh/hontest-habit/internal/app/auth/models"
	"github.com/insanelyharsh/hontest-habit/internal/webserver"
)

type AuthController struct {
	authManager *auth.AuthManager
}

func NewAuthController(authManager *auth.AuthManager) *AuthController {
	return &AuthController{
		authManager: authManager,
	}
}

// Routes mounts signup/login on group.
func (ac *AuthController) Routes(group webserver.Group) {
	group.POST("/signup", ac.handleSignup())
	group.POST("/login", ac.handleLogin())
}

func (ac *AuthController) handleSignup() webserver.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		var req models.SignupRequest
		if err := webserver.DecodeJSON(r, &req); err != nil {
			return err
		}

		resp, err := ac.authManager.Signup(r.Context(), &req)
		if err != nil {
			return err
		}

		webserver.WriteJSON(w, http.StatusCreated, resp)
		return nil
	}
}

func (ac *AuthController) handleLogin() webserver.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		var req models.LoginRequest
		if err := webserver.DecodeJSON(r, &req); err != nil {
			return err
		}

		resp, err := ac.authManager.Login(r.Context(), &req)
		if err != nil {
			return err
		}

		webserver.WriteJSON(w, http.StatusOK, resp)
		return nil
	}
}
