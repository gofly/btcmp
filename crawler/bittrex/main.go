package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gofly/btcmp/crawler"
	"github.com/gofly/btcmp/entity"
	scraper "github.com/yeouchien/go-cloudflare-scraper"
	"github.com/yeouchien/signalr"
)

const (
	vendor       = "Bittrex"
	SummariesAPI = "https://www.bittrex.com/api/v2.0/pub/Markets/GetMarketSummaries"
	WsCoreHub    = "CoreHub"
)

var (
	receiverHost string
)

func init() {
	flag.StringVar(&receiverHost, "receiver-host", "localhost:8082", "receiver host")
	flag.Parse()
}

type Orderb struct {
	Quantity float64 `json:"Quantity"`
	Rate     float64 `json:"Rate"`
}

type OrderUpdate struct {
	Orderb
	Type int
}

type Fill struct {
	Orderb
	OrderType string
}

// ExchangeState contains fills and order book updates for a pair.
type ExchangeState struct {
	MarketName string
	Nounce     int
	Buys       []OrderUpdate
	Sells      []OrderUpdate
	Fills      []Fill
	Initial    bool
}

func getPairSummaryTickers() (entity.Tickers, error) {
	tickers := make(entity.Tickers, 0)
	// transport, err := scraper.NewTransport(http.DefaultTransport)
	// if err != nil {
	// 	return nil, err
	// }
	// client := &http.Client{Transport: transport}
	client := http.DefaultClient
	req, _ := http.NewRequest(http.MethodGet, SummariesAPI, nil)
	req.Header.Set("User-Agent", scraper.UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data := struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Result  []struct {
			Summary struct {
				MarketName string
				Last       float64
			}
		}
	}{}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return nil, err
	}
	if !data.Success {
		return nil, fmt.Errorf("GetPairSummaries %s", data.Message)
	}
	for _, result := range data.Result {
		tickers = append(tickers, entity.Ticker{
			Vendor:   vendor,
			PairName: entity.PairName(result.Summary.MarketName),
			Last:     result.Summary.Last,
		})
	}
	return tickers, nil
}

func convertStates(statesCh <-chan []ExchangeState) <-chan entity.Tickers {
	tickersCh := make(chan entity.Tickers, 0)
	go func() {
		for states := range statesCh {
			tickers := make(entity.Tickers, 0, len(states))
			for _, state := range states {
				fillsLen := len(state.Fills)
				if fillsLen == 0 {
					continue
				}
				tickers = append(tickers, entity.Ticker{
					Vendor:   vendor,
					PairName: entity.PairName(state.MarketName),
					Last:     state.Fills[fillsLen-1].Rate,
				})
			}
			if len(tickers) > 0 {
				tickersCh <- tickers
			}
		}
	}()
	return tickersCh
}

func parseExchangeStates(messages []json.RawMessage) ([]ExchangeState, error) {
	states := make([]ExchangeState, 0)
	for _, msg := range messages {
		st := ExchangeState{}
		err := json.Unmarshal(msg, &st)
		if err != nil {
			return nil, err
		}
		states = append(states, st)
	}
	return states, nil
}

func connectBittrexWs(pairNames entity.PairNames) <-chan []ExchangeState {
	statesCh := make(chan []ExchangeState, 10)
	ws := signalr.NewWebsocketClient()
	ws.OnClientMethod = func(hub string, method string, messages []json.RawMessage) {
		if hub != "CoreHub" || method != "updateExchangeState" {
			return
		}
		states, err := parseExchangeStates(messages)
		if err != nil {
			log.Printf("[ERROR] parseExchangeStates error: %s\n", err)
			return
		}
		statesCh <- states
	}
	ws.OnMessageError = func(err error) {
		panic(err)
	}
	go func() {
	Connect:
		for {
			err := ws.Connect("https", "socket.bittrex.com", []string{WsCoreHub})
			if err != nil {
				log.Printf("[ERROR] connect ws error: %s", err)
				time.Sleep(time.Second * 5)
				continue
			}
			for _, pairName := range pairNames {
				_, err := ws.CallHub("CoreHub", "SubscribeToExchangeDeltas", pairName)
				if err != nil {
					log.Printf("[ERROR] CallHub %s error: %s", pairName, err)
					time.Sleep(time.Second * 5)
					continue Connect
				}
			}
			break
		}
	}()
	return statesCh
}

func main() {
	tickersCh := make(chan entity.Tickers, 10)
	receiverSrv := crawler.NewReceiverService(receiverHost, vendor)
	err := receiverSrv.DialWs()
	if err != nil {
		panic(err)
	}
	errCh := receiverSrv.SendTickers(tickersCh)
	go func() {
		for err := range errCh {
			receiverSrv.CloseWs()
			panic(err)
		}
	}()

	tickers, err := getPairSummaryTickers()
	if err != nil {
		panic(err)
	}
	tickersCh <- tickers

	statesCh := connectBittrexWs(tickers.PairNames())
	for tickers := range convertStates(statesCh) {
		log.Printf("[INFO] send tickers: %+v", tickers)
		tickersCh <- tickers
	}
}
