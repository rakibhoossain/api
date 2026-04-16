package websocket

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all for now
	},
}

func HandleLiveConnect(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Websocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	log.Println("New live connection established")

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Live connection closed: %v", err)
			return
		}
		
		// Echo for heartbeat / pong logic usually goes here
		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Printf("Failed to write message: %v", err)
			return
		}
	}
}
