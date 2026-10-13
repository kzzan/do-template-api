package repository

import (
	"context"
	"errors"
	"time"

	"github.com/samber/do/v2"

	v1 "do-template-api/api/v1"
	"do-template-api/internal/model"
)

const userColumns = "user_id, nickname, password, email, created_at, updated_at"

// UserRepository 用户数据访问接口（service 层依赖它做单测替身）。
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	Update(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, userId string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
}

func NewUserRepository(injector do.Injector) (UserRepository, error) {
	return &userRepository{
		Repository: do.MustInvoke[*Repository](injector),
	}, nil
}

type userRepository struct {
	*Repository
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	now := time.Now().UTC()
	user.CreatedAt, user.UpdatedAt = now, now
	query := r.DB(ctx).Rebind("INSERT INTO users (" + userColumns + ") VALUES (?, ?, ?, ?, ?, ?)")
	_, err := r.DB(ctx).ExecContext(ctx, query,
		user.UserID, user.Nickname, user.Password, user.Email, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *userRepository) Update(ctx context.Context, user *model.User) error {
	user.UpdatedAt = time.Now().UTC()
	query := r.DB(ctx).Rebind("UPDATE users SET nickname = ?, password = ?, email = ?, updated_at = ? WHERE user_id = ?")
	_, err := r.DB(ctx).ExecContext(ctx, query,
		user.Nickname, user.Password, user.Email, user.UpdatedAt, user.UserID,
	)
	return err
}

func (r *userRepository) GetByID(ctx context.Context, userId string) (*model.User, error) {
	return r.getOne(ctx, "user_id = ?", userId)
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return r.getOne(ctx, "email = ?", email)
}

func (r *userRepository) getOne(ctx context.Context, where string, arg any) (*model.User, error) {
	var user model.User
	query := r.DB(ctx).Rebind("SELECT " + userColumns + " FROM users WHERE " + where)
	err := r.DB(ctx).GetContext(ctx, &user, query, arg)
	if errors.Is(err, ErrRecordNotFound) {
		return nil, v1.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}
