package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all origins for now
	},
}

func (s *APIServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Unable to upgrade websocket:", err)
		return 
	}
	defer conn.Close()

	ch := s.store.Subscribe()
	defer s.store.Unsubscribe(ch)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return // client disconnected or closed
			}
		}
	}()

	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return // channel was closed
			}
			if err := conn.WriteJSON(event); err != nil {
				return // write failed, client gone
			}
		case <-done:
			return // client disconnected
		}
	}
}
