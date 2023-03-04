package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func main() {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/ws", handleWebSocket)
	log.Println("http server listening...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "hello world")
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	go func() {
		defer conn.Close()

		sourcePort := conn.RemoteAddr().(*net.TCPAddr).Port

		for {
			timestamp := time.Now().UnixMicro()

			message := fmt.Sprintf(`{"sourcePort": %d, "timestamp": %d}`, sourcePort, timestamp)
			log.Println(message)
			if err = conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
				fmt.Println(err)
				return
			}
			time.Sleep(100 * time.Millisecond)
		}

	}()
}
