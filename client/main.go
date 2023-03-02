package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	url := "ws://localhost:8080/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	var response struct {
		SourcePort int64 `json:"sourcePort"`
		Timestamp  int64 `json:"timestamp"`
	}

	done := make(chan struct{})
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	go func() {
		for {
			if err := conn.ReadJSON(&response); err != nil {
				fmt.Println(err)
				close(done)
				return
			}

			serverTime := time.UnixMicro(response.Timestamp)
			// localTime := time.Now()

			timeDiff := time.Since(serverTime).Microseconds()

			fmt.Printf("Server Source Port:%d Time difference: %v\n", response.SourcePort, timeDiff)
		}
	}()

	<-interrupt
	fmt.Println("Interrupt signal received, closing connection...")
	if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		fmt.Println(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second * 1):
	}
}
