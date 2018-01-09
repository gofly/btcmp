package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/gofly/btcmp/crawler"
	"github.com/gofly/btcmp/entity"
)

var (
	receiverHost string
	jar          http.CookieJar
	client       *http.Client
)

func init() {
	flag.StringVar(&receiverHost, "receiver-host", "localhost:8082", "receiver host")
	flag.Parse()
}

type GateMessage struct {
	Result   bool    `json:"result"`
	AskRate0 float64 `json:"ask_rate0"`
	BidRate0 float64 `json:"bid_rate0"`
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

	s := NewGateIOService("data.gate.io", "ws.gate.io")
	pairCodeMap, err := s.GetPairs()
	if err != nil {
		panic(err)
	}

	log.Printf("pairCodeMap: %+v", pairCodeMap)
	tickerChan := make(chan entity.Ticker)
	go func() {
		wsMap := make(map[entity.PairName]struct{})
		for {
			pairNames, err := sendSrv.GetAllAttendedPairNames()
			if err != nil {
				log.Printf("[ERROR] GetAllAttendedPairNames error: %s", err)
			} else {
				log.Printf("[INFO] GetAllAttendedPairNames result: %+v", pairNames)
				for _, pairName := range pairNames {
					if _, ok := wsMap[pairName]; ok {
						continue
					}
					if pair, ok := pairCodeMap[pairName]; ok {
						log.Printf("[INFO] Receive(%s, %s)", pairName, pair)
						err = s.ReceiveData(pairName, pair, tickerChan)
						if err != nil {
							log.Printf("[ERROR] receive data of %s error: %s", pairName, err)
						}
					} else {
						log.Printf("[INFO] %s not in pair-code map", pairName)
					}
					wsMap[pairName] = struct{}{}
				}
			}
			time.Sleep(time.Minute)
		}
	}()
	for ticker := range tickerChan {
		tickersCh <- entity.Tickers{ticker}
	}
}
