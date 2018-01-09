package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofly/btcmp/entity"
	"github.com/gorilla/websocket"
)

const (
	vendor = "Okex"
)

type OkexService struct {
	apiHost           string
	wsHost            string
	wsConn            *websocket.Conn
	pairNameSymbolMap map[entity.PairName][2]string
}

func NewOkexService(apiHost, wsHost string) *OkexService {
	return &OkexService{
		apiHost:           apiHost,
		wsHost:            wsHost,
		pairNameSymbolMap: make(map[entity.PairName][2]string),
	}
}

func (s *OkexService) WsConnect() (err error) {
	reqURL := s.wsHost + "/websocket"
	s.wsConn, _, err = websocket.DefaultDialer.Dial(reqURL, nil)
	go func() {
		for range time.Tick(time.Second * 5) {
			err = s.wsPing()
			if err != nil {
				log.Printf("[ERROR] ws ping error: %s", err)
			}
		}
	}()
	return err
}
func (s *OkexService) wsPing() error {
	return s.wsConn.WriteJSON(map[string]string{"event": "ping"})
}
func (s *OkexService) WsReceiveData() <-chan entity.Ticker {
	tickerCh := make(chan entity.Ticker)
	go func() {
		for {
			_, p, err := s.wsConn.ReadMessage()
			if err != nil {
				panic(err)
			}
			event := &struct {
				Event string `json:"event"`
			}{}
			err = json.Unmarshal(p, event)
			if err == nil {
				log.Printf("[INFO] event: %s", event.Event)
				continue
			}
			v := make([]struct {
				Base  string `json:"base"`
				Quote string `json:"quote"`
				Data  struct {
					Last string `json:"last"`
				} `json:"data"`
			}, 0)
			err = json.Unmarshal(p, &v)
			if err != nil {
				log.Printf("[WARN] json.Unmarshal to data error: %s, body: %s", err, string(p))
				continue
			}
			log.Printf("[INFO] data: %+v", v)
			for _, data := range v {
				if data.Quote == "" || data.Base == "" || data.Data.Last == "" {
					continue
				}
				last, _ := strconv.ParseFloat(data.Data.Last, 64)
				pairName := entity.PairName(strings.ToUpper(data.Quote + "-" + data.Base))
				tickerCh <- entity.Ticker{
					Vendor:   vendor,
					PairName: pairName,
					Last:     last,
				}
			}
		}
	}()
	return tickerCh
}

func (s *OkexService) WsSubscribe(pairName entity.PairName) error {
	symbol, ok := s.pairNameSymbolMap[pairName]
	if !ok {
		return errors.New("pair name invalid")
	}
	return s.wsConn.WriteJSON(map[string]interface{}{
		"event": "addChannel",
		"parameters": map[string]string{
			"base":    symbol[0],
			"binary":  "0",
			"product": "spot",
			"quote":   symbol[1],
			"type":    "ticker",
		},
	})
}

func (s *OkexService) GetTickers() (entity.Tickers, error) {
	reqURL := s.apiHost + "/v2/markets/tickers"
	resp, err := http.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	v := &struct {
		Code int `json:"code"`
		Data []struct {
			Last   string `json:"last"`
			Symbol string `json:"symbol"`
		}
		DetailMsg string `json:"detailMsg"`
		Msg       string `json:"msg"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(v)
	if err != nil {
		return nil, err
	}
	if v.Code != 0 {
		return nil, errors.New(v.Msg)
	}
	tickers := make(entity.Tickers, 0)
	for _, data := range v.Data {
		parts := strings.Split(data.Symbol, "_")
		if len(parts) != 2 {
			continue
		}
		last, _ := strconv.ParseFloat(data.Last, 64)
		pairName := entity.PairName(strings.ToUpper(parts[1] + "-" + parts[0]))
		if pn, ok := pairNameRewrite[pairName]; ok {
			pairName = pn
		}
		s.pairNameSymbolMap[pairName] = [2]string{parts[0], parts[1]}
		tickers = append(tickers, entity.Ticker{
			Vendor:   vendor,
			PairName: pairName,
			Last:     last,
		})
	}
	return tickers, nil
}
