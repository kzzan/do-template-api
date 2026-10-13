package main

import (
	"context"
	"flag"
	stdlog "log"
	"syscall"
	"time"

	"github.com/samber/do/v2"
	"github.com/spf13/viper"

	"do-template-api/internal/handler"
	"do-template-api/internal/repository"
	"do-template-api/internal/server"
	"do-template-api/internal/service"
	"do-template-api/pkg/config"
	"do-template-api/pkg/jwt"
	"do-template-api/pkg/log"
)

// basePackage 注册基础设施（配置/日志/JWT），其余各层自带 Package。
func basePackage(confPath string) func(do.Injector) {
	return do.Package(
		do.Lazy(func(i do.Injector) (*viper.Viper, error) {
			return config.NewConfig(confPath), nil
		}),
		do.Lazy(func(i do.Injector) (*log.Logger, error) {
			return log.NewLogger(do.MustInvoke[*viper.Viper](i)), nil
		}),
		do.Lazy(func(i do.Injector) (*jwt.JWT, error) {
			return jwt.NewJWT(do.MustInvoke[*viper.Viper](i)), nil
		}),
	)
}

func main() {
	confPath := flag.String("conf", "config/local.yml", "config path, eg: -conf ./config/local.yml")
	flag.Parse()

	container := do.New(
		basePackage(*confPath),
		repository.Package,
		service.Package,
		handler.Package,
		server.Package,
	)

	httpSrv := do.MustInvoke[*server.HTTPServer](container)
	logger := do.MustInvoke[*log.Logger](container)

	go func() {
		if err := httpSrv.Start(); err != nil {
			stdlog.Printf("http server exited: %v", err)
		}
	}()

	logger.Info().Str("http", "http://"+httpSrv.Addr).Msg("server started")

	// 关闭限时 10s
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, report := container.ShutdownOnSignalsWithContext(shutdownCtx, syscall.SIGINT, syscall.SIGTERM)
	if !report.Succeed {
		stdlog.Printf("shutdown report: %v", report)
	}
}
