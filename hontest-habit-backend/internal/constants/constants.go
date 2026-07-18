package constants

import "github.com/insanelyharsh/hontest-habit/internal/types"

const (
	// AppModeServer boots the full webserver (db + redis + http).
	AppModeServer types.AppMode = "server"
	// AppModeMigrator only runs pending DB migrations, then exits.
	AppModeMigrator types.AppMode = "migrator"
)
