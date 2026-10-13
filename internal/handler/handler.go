// Package handler HTTP 处理器：参数校验 + 响应组装，不含业务逻辑。
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	"do-template-api/pkg/jwt"
)

// Package 注册本层全部服务。
var Package = do.Package(
	do.Lazy(NewUserHandler),
	do.Lazy(NewUserGrpc),
)

// GetUserIDFromCtx 从 gin 上下文取出 JWT claims 中的用户 ID。
func GetUserIDFromCtx(ctx *gin.Context) string {
	claims, ok := ctx.Get("claims")
	if !ok {
		return ""
	}
	if c, ok := claims.(*jwt.MyCustomClaims); ok {
		return c.UserID
	}
	return ""
}
