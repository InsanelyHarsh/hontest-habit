package constants

import "github.com/insanelyharsh/hontest-habit/internal/types"

const (
	// AppModeServer boots the full webserver (db + redis + http).
	AppModeServer types.AppMode = "server"
	// AppModeMigrator only runs pending DB migrations, then exits.
	AppModeMigrator types.AppMode = "migrator"
)

const (
	// TraceIDContextKey holds the per-request trace ID set by middlewares.TraceID.
	TraceIDContextKey types.ContextKey = "trace_id"
	// ClaimsContextKey holds the authenticated auth.Claims set by middlewares.Authenticate.
	ClaimsContextKey types.ContextKey = "claims"
)

const (
	DailyFrequency   types.Frequency = "DL"
	WeeklyFrequency  types.Frequency = "WL"
	MonthlyFrequency types.Frequency = "MT"
)
