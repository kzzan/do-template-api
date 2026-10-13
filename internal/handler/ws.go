package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WsEcho WebSocket 预留端点（GET /ws）：升级连接后回显消息，客户端断开即退出。
// 依赖注入经 zerolog ctx 种子（见 middleware.RequestLogger）。
// ponytail: 回显 demo；业务化时按模块拆 hub/会话结构，并收紧 CheckOrigin
func WsEcho(ctx *gin.Context) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return // Upgrade 已写回错误响应
	}
	defer conn.Close()

	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if err := conn.WriteMessage(msgType, msg); err != nil {
			return
		}
	}
}
