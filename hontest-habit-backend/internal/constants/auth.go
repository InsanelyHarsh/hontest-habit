package constants

import "time"

const (
	// BcryptCost is the bcrypt work factor used to hash user passwords.
	BcryptCost = 12

	// MinPasswordLength is the minimum accepted plaintext password length.
	MinPasswordLength = 8

	// MaxPasswordLength guards against bcrypt silently truncating input
	// beyond 72 bytes.
	MaxPasswordLength = 72

	// JWTExpiry is the lifetime of an issued access token.
	JWTExpiry = 24 * time.Hour
)
