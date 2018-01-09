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

	okexSrv := NewOkexService("https://www.okex.com", "wss://okexcomreal.bafang.com:10441")

	tickers, err := okexSrv.GetTickers()
	if err != nil {
		panic(err)
	}
	tickersCh <- tickers

	err = okexSrv.WsConnect()
	if err != nil {
		panic(err)
	}

	for _, ticker := range tickers {
		err = okexSrv.WsSubscribe(ticker.PairName)
		if err != nil {
			panic(err)
		}
	}

	tickerCh := okexSrv.WsReceiveData()
	for ticker := range tickerCh {
		tickersCh <- entity.Tickers{ticker}
	}
}
