package connsock

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	EvTypeAggTrade = "@aggTrade"
	// Time to wait before attempting to reconnect
	reconnectWaitTime = 5 * time.Second
)

type socketProducer struct {
	outputChan    chan []byte
	urlConnection string
}

func NewSocketProduecer(outChan chan []byte, url string) *socketProducer {
	return &socketProducer{
		outputChan:    outChan,
		urlConnection: url,
	}
}

func (sp *socketProducer) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		// Check for main context cancellation before attempting to connect
		select {
		case <-ctx.Done():
			slog.Info("Producer shutting down.", "url", sp.urlConnection)
			return
		default:
		}

		slog.Info("Attempting to connect...", "url", sp.urlConnection)
		conn, _, err := websocket.DefaultDialer.Dial(sp.urlConnection, nil)
		if err != nil {
			slog.Error("Failed to connect, will retry...", "url", sp.urlConnection, "error", err)
			time.Sleep(reconnectWaitTime)
			continue // Go to the next iteration of the loop to retry connection
		}

		slog.Info("Connection established.", "url", sp.urlConnection)
		sp.setupPingHandler(conn)

		// Channel to signal an error from the reader goroutine
		errChan := make(chan error, 1)
		readerCtx, cancelReader := context.WithCancel(ctx)

		// Start the reader goroutine
		go func() {
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					// If there is any error, report it and exit the goroutine
					select {
					case errChan <- err:
					case <-readerCtx.Done():
					}
					return
				}

				// Forward the message
				select {
				case sp.outputChan <- msg:
				case <-readerCtx.Done():
					return
				}
			}
		}()

		// Supervise the connection
	superviseLoop:
		for {
			select {
			case <-ctx.Done():
				// Main service is shutting down
				slog.Info("Main context cancelled, closing connection.", "url", sp.urlConnection)
				cancelReader()
				conn.Close()
				return // Exit Start method
			case err := <-errChan:
				// Reader goroutine failed
				slog.Warn("Connection error, will reconnect.", "url", sp.urlConnection, "error", err)
				cancelReader()
				conn.Close()
				break superviseLoop // Exit superviseLoop to trigger reconnect
			}
		}

		// Wait a moment before trying to reconnect
		time.Sleep(reconnectWaitTime)
	}
}

func (sp *socketProducer) setupPingHandler(conn *websocket.Conn) {
	slog.Info("Setting up ping handler.", "url", sp.urlConnection)
	conn.SetPingHandler(func(appData string) error {
		err := conn.WriteControl(
			websocket.PongMessage,
			[]byte(appData),
			time.Now().Add(10*time.Second),
		)
		if err == nil {
			slog.Info("Successfully sent Pong.", "url", sp.urlConnection)
		}
		return err
	})
}
