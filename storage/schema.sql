-- do-template-api 初始化 schema（幂等，可重复执行）
-- user_id 由应用侧 uuid 生成，主键不依赖自增
-- 唯一约束内联进建表语句：MySQL 8 不支持 CREATE INDEX IF NOT EXISTS，
-- 内联 CONSTRAINT 是 sqlite / mysql / postgres 三方都可移植的写法
CREATE TABLE IF NOT EXISTS users (
    user_id    VARCHAR(36) PRIMARY KEY,
    nickname   VARCHAR(64)  NOT NULL DEFAULT '',
    password   VARCHAR(128) NOT NULL,
    email      VARCHAR(255) NOT NULL,
    created_at TIMESTAMP    NOT NULL,
    updated_at TIMESTAMP    NOT NULL,
    CONSTRAINT uq_users_email UNIQUE (email)
);
