package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gofly/btcmp/crawler"

	"github.com/gofly/btcmp/entity"
	"github.com/gorilla/websocket"
)

const (
	vendor         = "Binance"
	wsURLAPI       = "https://www.binance.com/exchange/public/mktdataWssUrl"
	wsURLAPISuffix = "/!miniTicker@arr@3000ms"
	productAPI     = "https://www.binance.com/exchange/public/product"
	userAgent      = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/62.0.3202.62 Safari/537.36"
)

var (
	receiverHost      string
	symbolPairNameMap = make(map[string]entity.PairName)
)

func init() {
	flag.StringVar(&receiverHost, "receiver-host", "localhost:8082", "receiver host")
	flag.Parse()
}

func loadSymbolPairNameMap() (entity.Tickers, error) {
	req, _ := http.NewRequest(http.MethodGet, productAPI, nil)
	req.Header.Set("User-Agent", userAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data := &struct {
		Products []struct {
			Symbol     string `json:"symbol"`
			QuoteAsset string `json:"quoteAsset"`
			BaseAsset  string `json:"baseAsset"`
			Close      string `json:"close"`
		} `json:"data"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(data)
	if err != nil {
		return nil, err
	}
	tickers := make(entity.Tickers, len(data.Products))
	for i, product := range data.Products {
		pairName := entity.PairName(product.QuoteAsset + "-" + product.BaseAsset)
		symbolPairNameMap[product.Symbol] = pairName
		last, _ := strconv.ParseFloat(product.Close, 64)
		tickers[i] = entity.Ticker{
			Vendor:   vendor,
			PairName: pairName,
			Last:     last,
		}
	}
	return tickers, nil
}

func getWsURL(suffix string) (string, error) {
	req, _ := http.NewRequest(http.MethodGet, wsURLAPI, nil)
	req.Header.Set("User-Agent", userAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body) + suffix, nil
}

func connectBinanceWs(wsURL string, tickersCh chan<- entity.Tickers) {
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{
		"User-Agent": {userAgent},
	})
	if err != nil {
		log.Printf("[ERROR] dial ws error: %s", err)
		time.Sleep(time.Second * 10)
	}
	for {
		v := make([]struct {
			Symbol string `json:"s"`
			Last   string `json:"c"`
		}, 0)
		err = ws.ReadJSON(&v)
		if err != nil {
			panic(err)
		}
		log.Printf("[INFO] data: %+v", v)
		tickers := make(entity.Tickers, 0)
		for _, t := range v {
			last, err := strconv.ParseFloat(t.Last, 64)
			if err != nil {
				continue
			}
			if pairName, ok := symbolPairNameMap[t.Symbol]; ok {
				tickers = append(tickers, entity.Ticker{
					Vendor:   vendor,
					PairName: pairName,
					Last:     last,
				})
			}
		}
		tickersCh <- tickers
	}
}

func main() {
	tickersCh := make(chan entity.Tickers)
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

	tickers, err := loadSymbolPairNameMap()
	if err != nil {
		panic(err)
	}
	tickersCh <- tickers

	wsURL, err := getWsURL(wsURLAPISuffix)
	if err != nil {
		panic(err)
	}
	log.Printf("[INFO] connect ws: %s", wsURL)
	connectBinanceWs(wsURL, tickersCh)
}
