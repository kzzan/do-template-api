// Package server 服务器装配：gin 引擎、中间件、路由与优雅启停。
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/spf13/viper"

	v1 "do-template-api/api/v1"
	"do-template-api/internal/handler"
	"do-template-api/internal/middleware"
	"do-template-api/internal/repository"
	"do-template-api/pkg/jwt"
	"do-template-api/pkg/log"
)

// Package 注册本层全部服务。
var Package = do.Package(
	do.Lazy(NewHTTPServer),
	do.Lazy(NewGRPCServer),
)

// HTTPServer 包装 *http.Server；依赖经 do:"" 标签由容器在构造期注入，
// 实现 ShutdownerWithContextAndError 参与容器优雅关闭。
type HTTPServer struct {
	Config      *viper.Viper         `do:""`
	Logger      *log.Logger          `do:""`
	JWT         *jwt.JWT             `do:""`
	DB          *repository.Database `do:""`
	UserHandler *handler.UserHandler `do:""`
	*http.Server
}

func NewHTTPServer(injector do.Injector) (*HTTPServer, error) {
	s := do.MustInvokeStruct[*HTTPServer](injector)

	if s.Config.GetString("env") == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(middleware.Recovery(), middleware.RequestLogger(s.Logger), middleware.CORS())

	engine.GET("/", func(ctx *gin.Context) {
		v1.HandleSuccess(ctx, gin.H{"message": "Welcome to do-template-api"})
	})
	engine.GET("/healthz", s.healthCheck)

	v1Group := engine.Group("/v1")
	{
		noAuthRouter := v1Group.Group("/")
		{
			noAuthRouter.POST("/register", s.UserHandler.Register)
			noAuthRouter.POST("/login", s.UserHandler.Login)
		}

		noStrictAuthRouter := v1Group.Group("/", middleware.NoStrictAuth(s.JWT))
		{
			noStrictAuthRouter.GET("/user", s.UserHandler.GetProfile)
		}

		strictAuthRouter := v1Group.Group("/", middleware.StrictAuth(s.JWT))
		{
			strictAuthRouter.PUT("/user", s.UserHandler.UpdateProfile)
		}
	}

	// WebSocket 预留端点（无鉴权 demo）
	engine.GET("/ws", handler.WsEcho)

	s.Server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.Config.GetString("http.host"), s.Config.GetInt("http.port")),
		Handler:      engine,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return s, nil
}

// healthCheck 直连 Database 依赖做 ping；
// 不在 struct 里保存 injector 运行时调用（官方文档反模式）。
func (s *HTTPServer) healthCheck(ctx *gin.Context) {
	checkCtx, cancel := context.WithTimeout(ctx.Request.Context(), 2*time.Second)
	defer cancel()

	if err := s.DB.HealthCheck(checkCtx); err != nil {
		v1.HandleError(ctx, http.StatusServiceUnavailable, v1.ErrInternalServerError, err)
		return
	}
	v1.HandleSuccess(ctx, nil)
}

// Start 阻塞监听；返回 http.ErrServerClosed 表示正常关闭。
func (s *HTTPServer) Start() error {
	if err := s.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown 实现 do.ShutdownerWithContextAndError，容器关闭时被调用。
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}
