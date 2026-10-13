package handler

import (
	"context"
	"time"

	"github.com/samber/do/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	userv1 "do-template-api/api/pb/userv1"
	"do-template-api/internal/service"
)

// UserGrpc 实现 userv1.UserServiceServer，作为 gRPC 预留入口的独立服务实现；
// 业务请求直接走 HTTP handler → service，不经 gRPC 中转。
// 内嵌 Unimplemented 保证 proto 未实现的方法返回 codes.Unimplemented。
type UserGrpc struct {
	userv1.UnimplementedUserServiceServer
	userService service.UserService `do:""`
}

func NewUserGrpc(injector do.Injector) (*UserGrpc, error) {
	return do.MustInvokeStruct[*UserGrpc](injector), nil
}

func (g *UserGrpc) GetProfile(ctx context.Context, req *userv1.GetProfileRequest) (*userv1.Profile, error) {
	profile, err := g.userService.GetProfile(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &userv1.Profile{UserId: profile.UserID, Nickname: profile.Nickname}, nil
}

// SubscribeHeartbeat 服务端流（长连接预留示例）：按 interval_ms 推送 count 次，
// 客户端断开（ctx 取消）即停止。
func (g *UserGrpc) SubscribeHeartbeat(req *userv1.SubscribeHeartbeatRequest, stream userv1.UserService_SubscribeHeartbeatServer) error {
	interval := time.Duration(req.GetIntervalMs()) * time.Millisecond
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	count := req.GetCount()
	if count <= 0 {
		count = 3
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for seq := int64(1); seq <= int64(count); seq++ {
		select {
		case <-stream.Context().Done():
			return nil
		case <-ticker.C:
			if err := stream.Send(&userv1.Heartbeat{Seq: seq, Timestamp: time.Now().UnixMilli()}); err != nil {
				return err
			}
		}
	}
	return nil
}
