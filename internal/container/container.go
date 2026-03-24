package container

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/username/banking-app/internal/config"
	appdb "github.com/username/banking-app/internal/db"
	"github.com/username/banking-app/internal/handler"
	"github.com/username/banking-app/internal/service"
)

type Container struct {
	Config       *config.Config
	Logger       *slog.Logger
	DBPool       *pgxpool.Pool
	HTTPHandler  http.Handler
	TransferSvc  service.TransferService
}

func Build(ctx context.Context) (*Container, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	logger := newLogger(cfg)

	dbPool, err := appdb.NewPool(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("init db pool: %w", err)
	}

	transferSvc := service.NewTransferService(dbPool, logger)
	transferHandler := handler.NewTransferHandler(transferSvc)
	router := handler.NewRouter(transferHandler)

	return &Container{
		Config:      cfg,
		Logger:      logger,
		DBPool:      dbPool,
		HTTPHandler: router,
		TransferSvc: transferSvc,
	}, nil
}

func (c *Container) Close() {
	if c == nil || c.DBPool == nil {
		return
	}
	c.DBPool.Close()
}

func newLogger(cfg *config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{}

	switch strings.ToLower(cfg.Logging.Level) {
	case "debug":
		opts.Level = slog.LevelDebug
	case "warn":
		opts.Level = slog.LevelWarn
	case "error":
		opts.Level = slog.LevelError
	default:
		opts.Level = slog.LevelInfo
	}

	if strings.ToLower(cfg.Logging.Format) == "text" {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}