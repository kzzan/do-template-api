package service

import (
	"context"
	"sync"
	"testing"

	"github.com/samber/do/v2"
	"github.com/spf13/viper"

	v1 "do-template-api/api/v1"
	"do-template-api/internal/model"
	"do-template-api/internal/repository"
	"do-template-api/pkg/jwt"
)

// fakeUserRepo 内存实现的用户仓库替身。
type fakeUserRepo struct {
	mu    sync.Mutex
	users map[string]*model.User // key: email
}

func (f *fakeUserRepo) Create(_ context.Context, user *model.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.users[user.Email] = user
	return nil
}

func (f *fakeUserRepo) Update(_ context.Context, user *model.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.users[user.Email] = user
	return nil
}

func (f *fakeUserRepo) GetByID(_ context.Context, userId string) (*model.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, u := range f.users {
		if u.UserID == userId {
			return u, nil
		}
	}
	return nil, v1.ErrNotFound
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.users[email]; ok {
		return u, nil
	}
	return nil, v1.ErrNotFound
}

// passthroughTx 测试用事务：直接执行 fn。
type passthroughTx struct{}

func (passthroughTx) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// newTestService 用 do 容器装配被测服务（官方推荐的测试方式）：
// 仓库与事务用假实现注入，JWT 用空配置真实现。
func newTestService(repo repository.UserRepository) UserService {
	injector := do.New()
	do.Provide(injector, func(do.Injector) (repository.UserRepository, error) { return repo, nil })
	do.Provide(injector, func(do.Injector) (repository.Transaction, error) { return passthroughTx{}, nil })
	do.Provide(injector, func(do.Injector) (*jwt.JWT, error) { return jwt.NewJWT(viper.New()), nil })
	do.Provide(injector, NewUserService)
	return do.MustInvoke[UserService](injector)
}

func TestRegisterAndLogin(t *testing.T) {
	svc := newTestService(&fakeUserRepo{users: map[string]*model.User{}})
	ctx := context.Background()

	if err := svc.Register(ctx, &v1.RegisterRequest{Email: "a@b.com", Password: "123456"}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	token, err := svc.Login(ctx, &v1.LoginRequest{Email: "a@b.com", Password: "123456"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	svc := newTestService(&fakeUserRepo{users: map[string]*model.User{}})
	ctx := context.Background()

	req := &v1.RegisterRequest{Email: "a@b.com", Password: "123456"}
	if err := svc.Register(ctx, req); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	err := svc.Register(ctx, req)
	if err != v1.ErrEmailAlreadyUse {
		t.Fatalf("second Register err = %v, want ErrEmailAlreadyUse", err)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	svc := newTestService(&fakeUserRepo{users: map[string]*model.User{}})
	ctx := context.Background()

	if err := svc.Register(ctx, &v1.RegisterRequest{Email: "a@b.com", Password: "123456"}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	tests := []struct {
		name string
		req  v1.LoginRequest
	}{
		{name: "wrong password", req: v1.LoginRequest{Email: "a@b.com", Password: "bad"}},
		{name: "unknown email", req: v1.LoginRequest{Email: "nobody@b.com", Password: "123456"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.Login(ctx, &tt.req); err != v1.ErrUnauthorized {
				t.Errorf("Login err = %v, want ErrUnauthorized", err)
			}
		})
	}
}

func TestGetAndUpdateProfile(t *testing.T) {
	repo := &fakeUserRepo{users: map[string]*model.User{}}
	svc := newTestService(repo)
	ctx := context.Background()

	if err := svc.Register(ctx, &v1.RegisterRequest{Email: "a@b.com", Password: "123456"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	userId := repo.users["a@b.com"].UserID

	profile, err := svc.GetProfile(ctx, userId)
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if profile.UserID != userId {
		t.Errorf("profile.UserID = %q, want %q", profile.UserID, userId)
	}

	if err := svc.UpdateProfile(ctx, userId, &v1.UpdateProfileRequest{Nickname: "alan", Email: "a@b.com"}); err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if got := repo.users["a@b.com"].Nickname; got != "alan" {
		t.Errorf("nickname = %q, want alan", got)
	}
}
