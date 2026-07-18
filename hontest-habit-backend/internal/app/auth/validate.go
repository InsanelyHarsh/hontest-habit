package auth

import (
	"fmt"
	"net/mail"
	"strings"

	"github.com/insanelyharsh/hontest-habit/internal/common/errors"
	"github.com/insanelyharsh/hontest-habit/internal/constants"
)

// validateEmail parses and normalizes raw, returning a lowercased bare
// address. net/mail.ParseAddress alone would accept "Display Name
// <a@b.com>" forms; the exact-match check against the trimmed input
// rejects those.
func validateEmail(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	addr, err := mail.ParseAddress(trimmed)
	if err != nil {
		return "", errors.BadRequest("invalid email address", err)
	}
	if addr.Address != trimmed {
		return "", errors.BadRequest("invalid email address", nil)
	}
	return strings.ToLower(addr.Address), nil
}

// validatePassword enforces plaintext length bounds. The upper bound
// exists solely to prevent silent bcrypt truncation.
func validatePassword(pw string) error {
	switch {
	case len(pw) < constants.MinPasswordLength:
		return errors.BadRequest(fmt.Sprintf("password must be at least %d characters", constants.MinPasswordLength), nil)
	case len(pw) > constants.MaxPasswordLength:
		return errors.BadRequest(fmt.Sprintf("password must be at most %d characters", constants.MaxPasswordLength), nil)
	}
	return nil
}
