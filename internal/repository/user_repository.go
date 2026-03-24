package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/username/banking-app/internal/db"
	"github.com/username/banking-app/internal/domain"
)

type userRepository struct {
	queries *db.Queries
	logger  *slog.Logger
}

func NewUserRepository(dbtx db.DBTX, logger *slog.Logger) UserRepository {
	if logger == nil {
		logger = slog.Default()
	}

	return &userRepository{
		queries: db.New(dbtx),
		logger:  logger,
	}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	if err := user.Validate(); err != nil {
		return err
	}

	created, err := r.queries.CreateUser(ctx, &db.CreateUserParams{
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		FullName:     user.FullName,
	})
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	*user = *toDomainUser(created)
	r.logger.InfoContext(ctx, "user created", "user_id", user.ID.String())
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row, err := r.queries.GetUserByID(ctx, toPgUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return toDomainUser(row), nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return toDomainUser(row), nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	if err := user.Validate(); err != nil {
		return err
	}

	updated, err := r.queries.UpdateUser(ctx, &db.UpdateUserParams{
		ID:       toPgUUID(user.ID),
		Email:    user.Email,
		FullName: user.FullName,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrUserNotFound
		}
		return fmt.Errorf("update user: %w", err)
	}

	*user = *toDomainUser(updated)
	r.logger.InfoContext(ctx, "user updated", "user_id", user.ID.String())
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteUser(ctx, toPgUUID(id)); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	r.logger.InfoContext(ctx, "user soft deleted", "user_id", id.String())
	return nil
}
