package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"golang.org/x/crypto/bcrypt"

	v1 "do-template-api/api/v1"
	"do-template-api/internal/model"
	"do-template-api/internal/repository"
	"do-template-api/pkg/jwt"
)

// UserService 用户业务接口。
type UserService interface {
	Register(ctx context.Context, req *v1.RegisterRequest) error
	Login(ctx context.Context, req *v1.LoginRequest) (string, error)
	GetProfile(ctx context.Context, userId string) (*v1.GetProfileResponseData, error)
	UpdateProfile(ctx context.Context, userId string, req *v1.UpdateProfileRequest) error
}

func NewUserService(injector do.Injector) (UserService, error) {
	return &userService{
		userRepo: do.MustInvoke[repository.UserRepository](injector),
		tm:       do.MustInvoke[repository.Transaction](injector),
		jwt:      do.MustInvoke[*jwt.JWT](injector),
	}, nil
}

type userService struct {
	userRepo repository.UserRepository
	tm       repository.Transaction
	jwt      *jwt.JWT
}

// Register 邮箱查重后创建用户；创建走事务，便于后续扩展多表写入。
func (s *userService) Register(ctx context.Context, req *v1.RegisterRequest) error {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, v1.ErrNotFound) {
		return v1.ErrInternalServerError
	}
	if user != nil {
		return v1.ErrEmailAlreadyUse
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user = &model.User{
		UserID:   uuid.NewString(),
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	return s.tm.Transaction(ctx, func(ctx context.Context) error {
		return s.userRepo.Create(ctx, user)
	})
}

func (s *userService) Login(ctx context.Context, req *v1.LoginRequest) (string, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return "", v1.ErrUnauthorized
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return "", v1.ErrUnauthorized
	}
	return s.jwt.GenToken(user.UserID, time.Now().Add(90*24*time.Hour))
}

func (s *userService) GetProfile(ctx context.Context, userId string) (*v1.GetProfileResponseData, error) {
	user, err := s.userRepo.GetByID(ctx, userId)
	if err != nil {
		return nil, err
	}
	return &v1.GetProfileResponseData{
		UserID:   user.UserID,
		Nickname: user.Nickname,
	}, nil
}

func (s *userService) UpdateProfile(ctx context.Context, userId string, req *v1.UpdateProfileRequest) error {
	user, err := s.userRepo.GetByID(ctx, userId)
	if err != nil {
		return err
	}
	user.Email = req.Email
	user.Nickname = req.Nickname
	return s.userRepo.Update(ctx, user)
}
