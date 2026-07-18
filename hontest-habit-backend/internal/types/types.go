package types

// AppMode selects which part of the application cmd/main.go boots into.
type AppMode string

// ContextKey is the type for values stored on a request context, so they
// can't collide with keys from other packages.
type ContextKey string

type UserId int64

type BlocklistId int64
type Frequency string
