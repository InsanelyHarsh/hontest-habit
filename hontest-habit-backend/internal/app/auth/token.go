package auth

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"

	"github.com/insanelyharsh/hontest-habit/internal/common/errors"
	"github.com/insanelyharsh/hontest-habit/internal/constants"
	"github.com/insanelyharsh/hontest-habit/internal/types"
)

// JWTConfig holds JWT signing parameters. Unlike db/redis Config, this is
// specific to the auth feature rather than a shared platform dependency,
// so it lives here rather than under internal/platform.
type JWTConfig struct {
	Secret string
	TTL    time.Duration
}

// JWTConfigFromEnv reads JWT_SECRET via viper.
func JWTConfigFromEnv() JWTConfig {
	return JWTConfig{
		Secret: viper.GetString("JWT_SECRET"),
		TTL:    constants.JWTExpiry,
	}
}

// Claims are the JWT claims issued on signup/login.
type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func generateJwtToken(ctx context.Context, cfg JWTConfig, userID types.UserId, email string) (string, error) {
	now := time.Now()
	claims := Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(int64(userID), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.TTL)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.Secret))
	if err != nil {
		return "", fmt.Errorf("auth: sign jwt: %w", err)
	}
	return signed, nil
}

// ValidateToken verifies tokenString against cfg and returns its claims.
func ValidateToken(ctx context.Context, cfg JWTConfig, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("auth: unexpected signing method %v", t.Header["alg"])
		}
		return []byte(cfg.Secret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.Unauthorized("invalid or expired token", err)
	}
	return claims, nil
}
