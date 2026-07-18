package repository

import (
	"context"
	goerrors "errors"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/insanelyharsh/hontest-habit/internal/common/errors"
	"github.com/insanelyharsh/hontest-habit/internal/constants"
	"github.com/insanelyharsh/hontest-habit/internal/platform/db"
)

// dummyBcryptHash is a precomputed bcrypt hash of a fixed, never-matching
// value. ValidateCredentials compares against it even when no user row is
// found, so an unknown email and a wrong password take comparable time and
// don't leak account existence via a timing side-channel.
const dummyBcryptHash = "$2a$12$CwTycUXWue0Thq9StjUM0uJ8vFN0R/n/nJ1CkNr1r4h5Yj2H3Xrk."

// AuthRepository owns all password crypto (hashing on create, comparison on
// validate) as well as persistence. No method returns a password hash to
// the caller — callers only ever get back a user ID.
type AuthRepository interface {
	// EmailExists reports whether email is already registered.
	EmailExists(ctx context.Context, email string) (bool, error)

	// CreateUser hashes plaintextPassword and inserts a new user row.
	// Returns an errors.Conflict if email is already registered.
	CreateUser(ctx context.Context, email, plaintextPassword string) (userID string, err error)

	// ValidateCredentials fetches the stored hash for email internally and
	// compares it against plaintextPassword. Returns errors.Unauthorized
	// (never a more specific reason) whether the email doesn't exist or
	// the password is wrong.
	ValidateCredentials(ctx context.Context, email, plaintextPassword string) (userID string, err error)
}

type AuthRepositoryImpl struct {
	db db.IDB
}

func NewAuthRepository(conn db.IDB) AuthRepository {
	return AuthRepositoryImpl{db: conn}
}

func (r AuthRepositoryImpl) EmailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`,
		email,
	).Scan(&exists)
	if err != nil {
		return false, errors.Internal("failed to check email existence", err)
	}
	return exists, nil
}

func (r AuthRepositoryImpl) CreateUser(ctx context.Context, email, plaintextPassword string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), constants.BcryptCost)
	if err != nil {
		return "", errors.Internal("failed to hash password", err)
	}

	var id string
	err = r.db.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) ON CONFLICT (email) DO NOTHING RETURNING id::text`,
		email, string(hash),
	).Scan(&id)
	switch {
	case err == nil:
		return id, nil
	case goerrors.Is(err, pgx.ErrNoRows):
		return "", errors.Conflict("email already registered", nil)
	default:
		return "", errors.Internal("failed to create user", err)
	}
}

func (r AuthRepositoryImpl) ValidateCredentials(ctx context.Context, email, plaintextPassword string) (string, error) {
	var id, hash string
	err := r.db.QueryRow(ctx,
		`SELECT id::text, password_hash FROM users WHERE email = $1`,
		email,
	).Scan(&id, &hash)

	switch {
	case err == nil:
		if compareErr := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintextPassword)); compareErr != nil {
			return "", errors.Unauthorized("invalid email or password", compareErr)
		}
		return id, nil
	case goerrors.Is(err, pgx.ErrNoRows):
		_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(plaintextPassword))
		return "", errors.Unauthorized("invalid email or password", nil)
	default:
		return "", errors.Internal("failed to fetch user", err)
	}
}
