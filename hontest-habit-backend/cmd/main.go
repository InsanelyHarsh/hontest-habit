package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/spf13/viper"

	"github.com/insanelyharsh/hontest-habit/internal/app/auth"
	"github.com/insanelyharsh/hontest-habit/internal/app/auth/repository"
	"github.com/insanelyharsh/hontest-habit/internal/app/blocklist"
	blocklistrepo "github.com/insanelyharsh/hontest-habit/internal/app/blocklist/repository"
	"github.com/insanelyharsh/hontest-habit/internal/constants"
	"github.com/insanelyharsh/hontest-habit/internal/platform/db"
	"github.com/insanelyharsh/hontest-habit/internal/platform/redis"
	"github.com/insanelyharsh/hontest-habit/internal/types"
	"github.com/insanelyharsh/hontest-habit/internal/webserver"
	"github.com/insanelyharsh/hontest-habit/internal/webserver/middlewares"
	"github.com/insanelyharsh/hontest-habit/internal/webserver/routes"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()
	viper.SetDefault("APP_MODE", string(constants.AppModeServer))
	if err := viper.ReadInConfig(); err != nil {
		slog.Warn("main: no .env file found, relying on process environment", "error", err)
	}

	mode := types.AppMode(viper.GetString("APP_MODE"))

	switch mode {
	case constants.AppModeMigrator:
		runMigrator()
	case constants.AppModeServer:
		runServer()
	default:
		slog.Error("main: unknown APP_MODE", "mode", mode)
		os.Exit(1)
	}
}

// runMigrator applies pending DB migrations, then exits.
func runMigrator() {
	dbCfg := db.ConfigFromEnv()
	if err := db.RunMigrations(dbCfg); err != nil {
		slog.Error("main: migrations failed", "error", err)
		os.Exit(1)
	}
	slog.Info("main: migrations complete")
}

// runServer connects to db/redis and starts the webserver.
func runServer() {
	ctx := context.Background()

	dbCfg := db.ConfigFromEnv()
	pool, err := db.New(ctx, dbCfg)
	if err != nil {
		slog.Error("main: db connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	rdb, err := redis.New(ctx, redis.ConfigFromEnv())
	if err != nil {
		slog.Error("main: redis connection failed", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	jwtCfg := auth.JWTConfigFromEnv()
	if jwtCfg.Secret == "" {
		slog.Error("main: JWT_SECRET is required")
		os.Exit(1)
	}
	authManager := auth.NewAuthManager(repository.NewAuthRepository(pool), jwtCfg)
	blocklistManager := blocklist.NewBlocklistManager(blocklistrepo.NewBlocklistRepository(pool))

	server := webserver.NewServer()
	server.Register("/auth/", routes.NewAuthController(authManager))
	server.Register("/blocklist/", routes.NewBlocklistController(blocklistManager), middlewares.Authenticate(jwtCfg))

	if err := server.InitWebServer(); err != nil {
		slog.Error("main: webserver exited", "error", err)
		os.Exit(1)
	}
}
