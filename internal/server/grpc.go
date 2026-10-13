package server

import (
	"context"
	"fmt"
	"net"

	"github.com/samber/do/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	userv1 "do-template-api/api/pb/userv1"
	"do-template-api/internal/handler"
	"do-template-api/pkg/log"

	"github.com/spf13/viper"
)

// GRPCServer 依赖经 do:"" 标签由容器在构造期注入；
// 实现 ShutdownerWithContextAndError 参与容器优雅关闭。
type GRPCServer struct {
	Config   *viper.Viper      `do:""`
	Logger   *log.Logger       `do:""`
	UserGrpc *handler.UserGrpc `do:""`

	server *grpc.Server
	addr   string
}

func NewGRPCServer(injector do.Injector) (*GRPCServer, error) {
	s := do.MustInvokeStruct[*GRPCServer](injector)

	s.addr = fmt.Sprintf("%s:%d", s.Config.GetString("grpc.host"), s.Config.GetInt("grpc.port"))
	s.server = grpc.NewServer()

	userv1.RegisterUserServiceServer(s.server, s.UserGrpc)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(s.server, healthSrv)

	reflection.Register(s.server)
	return s, nil
}

// Start 阻塞监听；listener 由 Start 持有，关闭经 Shutdown。
func (s *GRPCServer) Start() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.Logger.Info().Str("addr", s.addr).Msg("grpc server started")
	return s.server.Serve(lis)
}

// Addr 监听地址。
func (s *GRPCServer) Addr() string {
	return s.addr
}

// Shutdown 实现 do.ShutdownerWithContextAndError；
// GracefulStop 等待在途流结束，ctx 超时则强制 Stop 断开长连接。
func (s *GRPCServer) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		s.server.Stop()
		return ctx.Err()
	}
}
