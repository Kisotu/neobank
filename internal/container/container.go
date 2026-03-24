package container

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/username/banking-app/internal/auth"
	"github.com/username/banking-app/internal/config"
	appdb "github.com/username/banking-app/internal/db"
	"github.com/username/banking-app/internal/handler"
	"github.com/username/banking-app/internal/repository"
	"github.com/username/banking-app/internal/service"
)

type Container struct {
	Config      *config.Config
	Logger      *slog.Logger
	DBPool      *pgxpool.Pool
	HTTPHandler http.Handler
	JWTManager  *auth.JWTManager
	UserSvc     service.UserService
	AccountSvc  service.AccountService
	TransferSvc service.TransferService
	TxSvc       service.TransactionService
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

	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.Expiry, cfg.JWT.RefreshTTL)

	userRepo := repository.NewUserRepository(dbPool, logger)
	accountRepo := repository.NewAccountRepository(dbPool, logger)
	transactionRepo := repository.NewTransactionRepository(dbPool, logger)

	userSvc := service.NewUserService(userRepo, jwtManager, logger)
	accountSvc := service.NewAccountService(accountRepo, userRepo, logger)
	transferSvc := service.NewTransferService(dbPool, logger)
	txSvc := service.NewTransactionService(transactionRepo, accountRepo, logger)

	authHandler := handler.NewAuthHandler(userSvc, jwtManager)
	accountHandler := handler.NewAccountHandler(accountSvc)
	transferHandler := handler.NewTransferHandler(transferSvc)
	transactionHandler := handler.NewTransactionHandler(txSvc)
	router := handler.NewRouter(cfg, jwtManager, authHandler, accountHandler, transferHandler, transactionHandler)

	return &Container{
		Config:      cfg,
		Logger:      logger,
		DBPool:      dbPool,
		HTTPHandler: router,
		JWTManager:  jwtManager,
		UserSvc:     userSvc,
		AccountSvc:  accountSvc,
		TransferSvc: transferSvc,
		TxSvc:       txSvc,
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
