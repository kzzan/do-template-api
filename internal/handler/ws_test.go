package handler

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// TestWsEcho 验证 WebSocket 预留端点：升级、文本/二进制回显、正常关闭。
func TestWsEcho(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/ws", WsEcho)

	srv := httptest.NewServer(engine)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	tests := []struct {
		name    string
		msgType int
		payload string
	}{
		{name: "text", msgType: websocket.TextMessage, payload: "hello"},
		{name: "binary", msgType: websocket.BinaryMessage, payload: "\x00\x01\x02"},
		{name: "utf8 中文", msgType: websocket.TextMessage, payload: "你好，世界"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := conn.WriteMessage(tt.msgType, []byte(tt.payload)); err != nil {
				t.Fatalf("write: %v", err)
			}
			gotType, got, err := conn.ReadMessage()
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if gotType != tt.msgType || string(got) != tt.payload {
				t.Errorf("echo = (%d, %q), want (%d, %q)", gotType, got, tt.msgType, tt.payload)
			}
		})
	}

	if err := conn.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second)); err != nil {
		t.Fatalf("close: %v", err)
	}
}
