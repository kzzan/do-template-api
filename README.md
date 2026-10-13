# do-template-api

单体 Go 后端模板：**gin + sqlx + zerolog + samber/do/v2**（+ gRPC / WebSocket / redis 预留），
结构参考 [nunu-layout-advanced](../nunu-layout-advanced)、do 用法对齐 [samber/do-template-api](https://github.com/samber/do-template-api)。

## 快速开始

```bash
make migration   # 初始化 sqlite 表结构（storage/schema.sql）
make run         # 启动，默认 http://127.0.0.1:8000

# 可选：本地基础设施（redis 默认启用；postgres/mysql 按 profile）
make up                          # docker compose up -d（redis）
docker compose --profile mysql up -d
docker compose --profile postgres up -d

# 注册 / 登录 / 查看个人信息
curl -X POST localhost:8000/v1/register -H 'Content-Type: application/json' \
  -d '{"email":"a@b.com","password":"123456"}'
curl -X POST localhost:8000/v1/login -H 'Content-Type: application/json' \
  -d '{"email":"a@b.com","password":"123456"}'
curl localhost:8000/v1/user -H "Authorization: Bearer <accessToken>"
```

## 目录结构

```
cmd/server/       入口：do.New(packages...) 装配 + 信号等待
cmd/migration/    执行 storage/schema.sql（幂等）
api/proto/userv1/ proto 源文件（make proto 重新生成）
api/pb/userv1/    protoc 生成的 pb 代码
api/v1/           HTTP 请求/响应 DTO 与业务错误码
internal/
  handler/        HTTP 处理器 + gRPC 服务实现（UserGrpc，预留）+ WebSocket（WsEcho，预留）
  service/        业务逻辑
  repository/     sqlx 数据访问 + redis（懒加载）+ ctx 事务传播
  model/          数据模型（db 标签）
  middleware/     Recovery / 访问日志 / CORS / JWT
  server/         http.go（gin + /ws）+ grpc.go（gRPC 预留）+ 优雅启停
pkg/              config(viper) / log(zerolog) / jwt
config/           local.yml / prod.yml
storage/          sqlite 数据文件 + schema.sql
docker-compose.yml  本地基础设施（redis 默认；postgres/mysql 按 profile）
```

## gRPC 与 WebSocket（预留入口）

- proto：`api/proto/userv1/user.proto`，`make proto` 重新生成（需 protoc + protoc-gen-go/go-grpc）
- 服务端：`server.GRPCServer`（默认 :9500），注册 `userv1.UserService` + 标准 health + reflection（可用 grpcurl 直连调试）
- **一元**：`GetProfile` 复用 service 层查用户
- **长连接**：`SubscribeHeartbeat` 服务端流，按 `interval_ms` 推送 `count` 次，客户端断开即停
- **WebSocket**：`GET /ws` 回显端点（gorilla/websocket），升级/读写/关闭全链路已验证

## 请求链路

**业务主链路**：路由（server 内联注册）→ middleware → handler → service → repository → DB。
gRPC 与 WebSocket 为**预留入口**，不参与业务主链路：

- **gRPC 预留**（`server.GRPCServer`，默认 :9500）：独立入口供外部 gRPC 调用方直连；
  `grpc.enabled: false` 可整体关闭（不占端口、不实例化）；
  proto 在 `api/proto/userv1/`（`make proto` 重新生成），`handler.UserGrpc` 复用 service 层；
  已含一元（GetProfile）与服务端流（SubscribeHeartbeat 长连接示例）、标准 health、reflection（grpcurl 可直连）
- **WebSocket 预留**（`GET /ws`）：gorilla/websocket 回显端点，验证升级/读写/关闭全链路；
  gorilla Upgrade 会清除 conn deadline，30s HTTP 超时不影响长连接；业务化时拆 hub/会话并收紧 CheckOrigin

## 依赖注入（samber/do v2，官方 do-template-api 用法）

- **模块化注册**：各层自带 `Package`（`do.Package` + `do.Lazy`），`cmd/server/main.go` 用
  `do.New(base, repository.Package, service.Package, handler.Package, server.Package)` 一次装配
- **构造器签名**：图节点构造器统一 `NewX(injector do.Injector) (T, error)`，内部 `do.MustInvoke` 取依赖；
  叶子构造器（config/log/jwt）保持纯净，在 main 里闭包包装
- **标签注入**：`UserHandler`、`HTTPServer` 用 `do.MustInvokeStruct` + `do:""` 结构体标签自动装配
- **接口别名**：`do.Bind[*Repository, Transaction]()` 把同一实例以事务接口注册
- **生命周期**：`Database`/`HTTPServer` 实现 `Shutdowner`，`container.ShutdownOnSignals` 收到
  SIGINT/SIGTERM 后按依赖序优雅关闭；`Database.HealthCheck` 直连注入到 `/healthz`（不存 injector 运行时调用）

## 事务模式

service 层调用 `tm.Transaction(ctx, fn)`；fn 内所有 repo 方法通过 `Repository.DB(ctx)`
自动路由到同一事务（ctx 携带 *sqlx.Tx），回滚时错误用 `errors.Join` 合并。

## 切换数据库

`config/*.yml` 中改 `data.db.driver/dsn`（sqlite / mysql / postgres 均已内置驱动），
占位符跨库兼容（`Rebind` 自动转换 `?` → `$n`）。

## Redis

懒加载注册（`do.Lazy(NewRedis)`）：业务侧需要时 `do.MustInvoke[*repository.Redis]`
即建立连接（构造时 ping 验证）；未被引用则完全不连接。实现 `Shutdown`/`HealthCheck`，
实例化后随容器优雅关闭。地址在 `config/*.yml` 的 `data.redis` 配置。

## 测试

```bash
make test
```

- `pkg/jwt`：签发/解析往返、过期、非法 token（表驱动）
- `internal/service`：内存假仓库 + 假事务经 do 容器装配，覆盖注册/登录/资料读写
- `internal/repository`：sqlite `:memory:` 真库 CRUD 与事务回滚；redis 装配测试（本机无 redis 自动跳过）
