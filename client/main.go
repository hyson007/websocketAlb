package main

import (
	"fmt"
	"math"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

func main() {
	var url string
	var port int
	var maxNumber int

	rootCmd := &cobra.Command{
		Use:   "websocket-client",
		Short: "A WebSocket client that measures server-to-client latency",
		RunE: func(cmd *cobra.Command, args []string) error {
			connURL := fmt.Sprintf("ws://%s:%d/ws", url, port)
			conn, _, err := websocket.DefaultDialer.Dial(connURL, nil)
			if err != nil {
				return err
			}
			defer conn.Close()

			var response struct {
				SourcePort int64 `json:"sourcePort"`
				Timestamp  int64 `json:"timestamp"`
			}

			done := make(chan struct{})
			interrupt := make(chan os.Signal, 1)
			signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
			numberToPrint := 10
			count := 0
			var sumServerTime int64 = 0
			timeDiffs := make([]int64, 0, maxNumber)

			go func() {
				for i := 0; i < maxNumber; i++ {
					if err := conn.ReadJSON(&response); err != nil {
						fmt.Println(err)
						close(done)
						return
					}

					serverTime := time.UnixMicro(response.Timestamp)

					timeDiff := time.Since(serverTime).Microseconds()
					sumServerTime += timeDiff
					count++
					timeDiffs = append(timeDiffs, timeDiff)

					if count == numberToPrint {
						fmt.Printf("Server Source Port:%d, average server to client latency: %v\n", response.SourcePort, sumServerTime/int64(numberToPrint))
						count = 0
						sumServerTime = 0
					}
				}
				close(done)
			}()
		out:
			for {
				select {
				case <-interrupt:
					fmt.Println("Interrupt signal received, closing connection...")
					if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
						fmt.Println(err)
					}
					break out
				case <-done:
					fmt.Println("Reached max number of tests, closing connection...")
					if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
						fmt.Println(err)
					}

					break out
				}
			}
			sort.Slice(timeDiffs, func(i, j int) bool { return timeDiffs[i] < timeDiffs[j] })
			fmt.Printf("5th percentile: %v\n", percentile(5, timeDiffs))
			fmt.Printf("10th percentile: %v\n", percentile(10, timeDiffs))
			fmt.Printf("25th percentile: %v\n", percentile(25, timeDiffs))
			fmt.Printf("50th percentile: %v\n", percentile(50, timeDiffs))
			fmt.Printf("70th percentile: %v\n", percentile(70, timeDiffs))
			fmt.Printf("90th percentile: %v\n", percentile(90, timeDiffs))
			return nil
		},
	}

	rootCmd.Flags().StringVarP(&url, "url", "u", "localhost", "the URL of the WebSocket server")
	rootCmd.Flags().IntVarP(&port, "port", "p", 8080, "the port number of the WebSocket server")
	rootCmd.Flags().IntVarP(&maxNumber, "maxNumber", "m", 2000, "default number of times to test the latency")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}

func percentile(p int, xs []int64) int64 {
	if p <= 0 {
		return xs[0]
	}
	if p >= 100 {
		return xs[len(xs)-1]
	}
	rank := float64(p) / 100 * float64(len(xs)-1)
	lo := int(math.Floor(rank))
	hi := int(math.Ceil(rank))
	return (xs[lo]*(int64(hi)-int64(rank)) + xs[hi]*(int64(rank)-int64(lo))) / int64(hi-lo)
}
