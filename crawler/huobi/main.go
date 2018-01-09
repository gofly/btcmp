package main

import (
	"flag"

	"github.com/gofly/btcmp/crawler"
	"github.com/gofly/btcmp/entity"
)

var (
	receiverHost string
)

func init() {
	flag.StringVar(&receiverHost, "receiver-host", "localhost:8082", "receiver host")
	flag.Parse()
}
func main() {
	tickersCh := make(chan entity.Tickers, 10)
	sendSrv := crawler.NewReceiverService(receiverHost, vendor)
	err := sendSrv.DialWs()
	if err != nil {
		panic(err)
	}
	errCh := sendSrv.SendTickers(tickersCh)
	go func() {
		for err := range errCh {
			sendSrv.CloseWs()
			panic(err)
		}
	}()

	huobiSrv, err := NewHuobiService("https://api.huobi.pro", "wss://api.huobi.pro/ws")
	if err != nil {
		panic(err)
	}
	tickerCh, err := huobiSrv.WsReceiveTicker()
	if err != nil {
		panic(err)
	}
	for ticker := range tickerCh {
		tickersCh <- entity.Tickers{ticker}
	}
}
