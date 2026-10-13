package handler

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	userv1 "do-template-api/api/pb/userv1"
	v1 "do-template-api/api/v1"
	"do-template-api/internal/service"
)

// fakeUserService 仅实现 GetProfile 的最小替身（嵌入 nil 接口兜底其余方法）。
type fakeUserService struct {
	service.UserService
	profile *v1.GetProfileResponseData
}

func (f fakeUserService) GetProfile(context.Context, string) (*v1.GetProfileResponseData, error) {
	if f.profile == nil {
		return nil, v1.ErrNotFound
	}
	return f.profile, nil
}

// newBufconnGRPC 在内存通道上启动真实 grpc.Server 并返回客户端连接。
func newBufconnGRPC(t *testing.T, svc userv1.UserServiceServer) userv1.UserServiceClient {
	t.Helper()
	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	userv1.RegisterUserServiceServer(srv, svc)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc client: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return userv1.NewUserServiceClient(conn)
}

func TestUserGrpcGetProfile(t *testing.T) {
	client := newBufconnGRPC(t, &UserGrpc{userService: fakeUserService{
		profile: &v1.GetProfileResponseData{UserID: "u-1", Nickname: "alan"},
	}})

	profile, err := client.GetProfile(t.Context(), &userv1.GetProfileRequest{UserId: "u-1"})
	if err != nil {
		t.Fatalf("GetProfile: %v", err)
	}
	if profile.GetUserId() != "u-1" || profile.Nickname != "alan" {
		t.Errorf("profile = %+v", profile)
	}
}

func TestUserGrpcGetProfileNotFound(t *testing.T) {
	client := newBufconnGRPC(t, &UserGrpc{userService: fakeUserService{}})

	if _, err := client.GetProfile(t.Context(), &userv1.GetProfileRequest{UserId: "missing"}); err == nil {
		t.Error("expected error for missing user")
	}
}

// TestUserGrpcSubscribeHeartbeat 验证服务端流（长连接）：按序收到 count 次 Heartbeat。
func TestUserGrpcSubscribeHeartbeat(t *testing.T) {
	client := newBufconnGRPC(t, &UserGrpc{userService: fakeUserService{}})

	stream, err := client.SubscribeHeartbeat(t.Context(), &userv1.SubscribeHeartbeatRequest{
		Count:      3,
		IntervalMs: 10,
	})
	if err != nil {
		t.Fatalf("SubscribeHeartbeat: %v", err)
	}

	var seqs []int64
	for {
		beat, err := stream.Recv()
		if err != nil {
			break
		}
		seqs = append(seqs, beat.Seq)
		if beat.Timestamp == 0 {
			t.Error("timestamp should be set")
		}
	}
	if len(seqs) != 3 || seqs[0] != 1 || seqs[1] != 2 || seqs[2] != 3 {
		t.Errorf("seqs = %v, want [1 2 3]", seqs)
	}
}
