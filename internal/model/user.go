package model

import "time"

// User 数据库模型，db 标签对应 users 表列名。
type User struct {
	UserID    string    `db:"user_id"`
	Nickname  string    `db:"nickname"`
	Password  string    `db:"password"`
	Email     string    `db:"email"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
