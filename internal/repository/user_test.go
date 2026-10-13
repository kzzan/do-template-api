package repository

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"
	_ "modernc.org/sqlite"

	v1 "do-template-api/api/v1"
	"do-template-api/internal/model"
	"do-template-api/pkg/log"
)

const testSchema = `
CREATE TABLE IF NOT EXISTS users (
    user_id    TEXT PRIMARY KEY,
    nickname   TEXT NOT NULL DEFAULT '',
    password   TEXT NOT NULL,
    email      TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);
`

func newTestRepo(t *testing.T) (*Repository, func()) {
	t.Helper()
	db, err := sqlx.Connect("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}
	if _, err := db.Exec(testSchema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	logger := log.NewLogger(viper.New())
	r := &Repository{db: &Database{DB: db}, logger: logger}
	return r, func() { db.Close() }
}

func TestUserRepositoryCRUD(t *testing.T) {
	r, cleanup := newTestRepo(t)
	defer cleanup()

	userRepo := &userRepository{Repository: r}
	ctx := context.Background()

	user := &model.User{UserID: "u-1", Nickname: "alan", Password: "hashed", Email: "a@b.com"}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := userRepo.GetByID(ctx, "u-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Email != "a@b.com" || got.Nickname != "alan" {
		t.Errorf("got = %+v", got)
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set by Create")
	}

	byEmail, err := userRepo.GetByEmail(ctx, "a@b.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if byEmail.UserID != "u-1" {
		t.Errorf("byEmail.UserID = %q, want u-1", byEmail.UserID)
	}

	got.Nickname = "bob"
	if err := userRepo.Update(ctx, got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	updated, _ := userRepo.GetByID(ctx, "u-1")
	if updated.Nickname != "bob" {
		t.Errorf("nickname = %q, want bob", updated.Nickname)
	}
	if !updated.UpdatedAt.After(updated.CreatedAt) && updated.UpdatedAt.Equal(updated.CreatedAt) == false {
		t.Error("UpdatedAt should be refreshed")
	}
}

func TestGetNotFound(t *testing.T) {
	r, cleanup := newTestRepo(t)
	defer cleanup()

	userRepo := &userRepository{Repository: r}
	if _, err := userRepo.GetByID(context.Background(), "missing"); err != v1.ErrNotFound {
		t.Errorf("GetByID missing err = %v, want ErrNotFound", err)
	}
	if _, err := userRepo.GetByEmail(context.Background(), "missing@b.com"); err != v1.ErrNotFound {
		t.Errorf("GetByEmail missing err = %v, want ErrNotFound", err)
	}
}

func TestTransactionRollback(t *testing.T) {
	r, cleanup := newTestRepo(t)
	defer cleanup()

	userRepo := &userRepository{Repository: r}
	ctx := context.Background()

	// 提交/回滚双路径：fn 返回 error 时写入应回滚
	if err := r.Transaction(ctx, func(ctx context.Context) error {
		return userRepo.Create(ctx, &model.User{UserID: "u-ok", Email: "ok@b.com", Password: "p"})
	}); err != nil {
		t.Fatalf("Transaction commit: %v", err)
	}
	if _, err := userRepo.GetByEmail(ctx, "ok@b.com"); err != nil {
		t.Errorf("committed row invisible: %v", err)
	}

	err := r.Transaction(ctx, func(ctx context.Context) error {
		_ = userRepo.Create(ctx, &model.User{UserID: "u-rb", Email: "rb@b.com", Password: "p"})
		return context.DeadlineExceeded
	})
	if err == nil {
		t.Fatal("Transaction should propagate fn error")
	}
	if _, err := userRepo.GetByEmail(ctx, "rb@b.com"); err != v1.ErrNotFound {
		t.Errorf("rolled-back row visible: err = %v, want ErrNotFound", err)
	}
}

// TestTransactionNested 验证嵌套事务为扁平语义：内层加入外层，
// 外层回滚时内层写入一并回滚。
func TestTransactionNested(t *testing.T) {
	r, cleanup := newTestRepo(t)
	defer cleanup()

	userRepo := &userRepository{Repository: r}
	ctx := context.Background()

	if err := r.Transaction(ctx, func(ctx context.Context) error {
		if err := userRepo.Create(ctx, &model.User{UserID: "u-outer", Email: "outer@b.com", Password: "p"}); err != nil {
			return err
		}
		return r.Transaction(ctx, func(ctx context.Context) error {
			return userRepo.Create(ctx, &model.User{UserID: "u-inner", Email: "inner@b.com", Password: "p"})
		})
	}); err != nil {
		t.Fatalf("nested transaction: %v", err)
	}
	for _, email := range []string{"outer@b.com", "inner@b.com"} {
		if _, err := userRepo.GetByEmail(ctx, email); err != nil {
			t.Errorf("committed row %s invisible: %v", email, err)
		}
	}

	// 外层回滚 → 内层写入一并回滚
	if err := r.Transaction(ctx, func(ctx context.Context) error {
		_ = userRepo.Create(ctx, &model.User{UserID: "u-n1", Email: "n1@b.com", Password: "p"})
		return r.Transaction(ctx, func(ctx context.Context) error {
			_ = userRepo.Create(ctx, &model.User{UserID: "u-n2", Email: "n2@b.com", Password: "p"})
			return context.DeadlineExceeded
		})
	}); err == nil {
		t.Fatal("outer transaction should fail with inner error")
	}
	for _, email := range []string{"n1@b.com", "n2@b.com"} {
		if _, err := userRepo.GetByEmail(ctx, email); err != v1.ErrNotFound {
			t.Errorf("nested row %s should roll back with outer: err = %v", email, err)
		}
	}
}
