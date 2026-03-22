package api

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all origins for now
	},
}

func (s *APIServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 1. upgrade HTTP connection to WebSocket
	// 2. defer conn.Close()
	// 3. subscribe to store events
	// 4. defer unsubscribe
	// 5. loop: read from channel, write JSON to websocket
	// 6. exit when channel is closed or write fails
}
