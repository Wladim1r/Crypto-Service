package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"

	"github.com/Wladim1r/aggregator/converting"
	// "github.com/Wladim1r/aggregator/kaffka"
	"github.com/Wladim1r/aggregator/models"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	wg := new(sync.WaitGroup)

	rawMsgsChan := make(chan []byte, 100)
	secondStatChan := make(chan models.SecondStat, 100)
	// dailyStatChan := make(chan models.DailyStat, 500)
	// kafkaMsgChan := make(chan models.KafkaMsg, 500)

	// cfg := kaffka.LoadKafkaConfig()
	// producer := kaffka.NewProducer(cfg)

	wg.Add(2)
	go converting.ReceiveAggTradeMessage(ctx, wg, rawMsgsChan)
	// go converting.ReceiveMiniTickerMessage(ctx, wg, rawMsgsChan)
	// go converting.ConvertRawToArrDS(ctx, wg, rawMsgsChan, dailyStatChan)
	go converting.ConvertRawToArrSS(ctx, wg, rawMsgsChan, secondStatChan)
	// go converting.ReceiveKafkaMsg(ctx, wg, dailyStatChan, kafkaMsgChan)
	// go producer.Start(ctx, wg, kafkaMsgChan)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-secondStatChan:
				fmt.Println("==========================")
				fmt.Println(msg)
				fmt.Println("==========================")
			}
		}
	}()

	<-c
	cancel()
	slog.Info("👾 Received Interruption signal")
	slog.Info("⏲️ Wait for finishing all the goroutines...")
	wg.Wait()
	slog.Info("🏁 It is over 😢")
}
