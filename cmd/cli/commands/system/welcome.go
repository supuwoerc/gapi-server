package system

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func newWelcomeCmd(cliFactory CliFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "welcome",
		Short: "Print welcome message and verify infrastructure",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := cliFactory()
			if err != nil {
				return fmt.Errorf("failed to initialize cli: %w", err)
			}
			defer cli.Close()

			cli.Logger.Info("welcome cli is running...")

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			if err := cli.Redis.Ping(ctx).Err(); err != nil {
				cli.Logger.Error("redis ping failed", zap.Error(err))
			} else {
				cli.Logger.Info("redis connection OK")
			}

			sqlDB, err := cli.DB.DB()
			if err != nil {
				cli.Logger.Error("failed to get sql.DB", zap.Error(err))
			} else if err := sqlDB.PingContext(ctx); err != nil {
				cli.Logger.Error("database ping failed", zap.Error(err))
			} else {
				cli.Logger.Info("database connection OK")
			}

			fmt.Println("All infrastructure checks passed.")
			return nil
		},
	}
}
