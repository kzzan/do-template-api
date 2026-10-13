package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/samber/do/v2"

	v1 "do-template-api/api/v1"
	"do-template-api/internal/service"
)

// UserHandler 依赖通过 do:"" 标签由容器自动注入。
type UserHandler struct {
	userService service.UserService `do:""`
}

func NewUserHandler(injector do.Injector) (*UserHandler, error) {
	return do.MustInvokeStruct[*UserHandler](injector), nil
}

// Register 用户注册
func (h *UserHandler) Register(ctx *gin.Context) {
	req := new(v1.RegisterRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	if err := h.userService.Register(ctx, req); err != nil {
		zerolog.Ctx(ctx.Request.Context()).Error().Err(err).Msg("userService.Register")
		v1.HandleError(ctx, http.StatusInternalServerError, err, nil)
		return
	}
	v1.HandleSuccess(ctx, nil)
}

// Login 账号登录
func (h *UserHandler) Login(ctx *gin.Context) {
	var req v1.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	token, err := h.userService.Login(ctx, &req)
	if err != nil {
		v1.HandleError(ctx, http.StatusUnauthorized, v1.ErrUnauthorized, nil)
		return
	}
	v1.HandleSuccess(ctx, v1.LoginResponseData{AccessToken: token})
}

// GetProfile 获取用户信息
func (h *UserHandler) GetProfile(ctx *gin.Context) {
	userId := GetUserIDFromCtx(ctx)
	if userId == "" {
		v1.HandleError(ctx, http.StatusUnauthorized, v1.ErrUnauthorized, nil)
		return
	}
	user, err := h.userService.GetProfile(ctx, userId)
	if err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	v1.HandleSuccess(ctx, user)
}

// UpdateProfile 修改用户信息
func (h *UserHandler) UpdateProfile(ctx *gin.Context) {
	userId := GetUserIDFromCtx(ctx)
	if userId == "" {
		v1.HandleError(ctx, http.StatusUnauthorized, v1.ErrUnauthorized, nil)
		return
	}
	var req v1.UpdateProfileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		v1.HandleError(ctx, http.StatusBadRequest, v1.ErrBadRequest, nil)
		return
	}
	if err := h.userService.UpdateProfile(ctx, userId, &req); err != nil {
		zerolog.Ctx(ctx.Request.Context()).Error().Err(err).Msg("userService.UpdateProfile")
		v1.HandleError(ctx, http.StatusInternalServerError, v1.ErrInternalServerError, nil)
		return
	}
	v1.HandleSuccess(ctx, nil)
}
