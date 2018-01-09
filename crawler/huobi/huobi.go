package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gofly/btcmp/entity"
	"github.com/gorilla/websocket"
)

const vendor = "Huobi"

func gunzip(r io.Reader) (*bytes.Buffer, error) {
	r, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)
	_, err = buf.ReadFrom(r)
	return buf, err
}

type transport struct{}

func (t *transport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.71 Safari/537.36")
	return http.DefaultTransport.RoundTrip(r)
}

type symbol struct {
	BaseCurrency  string `json:"base-currency"`
	QuoteCurrency string `json:"quote-currency"`
	Symbol        string `json:"symbol"`
}

func (s symbol) PairName() entity.PairName {
	return entity.PairName(strings.ToUpper(s.QuoteCurrency + "-" + s.BaseCurrency))
}

type HuobiService struct {
	apiHost           string
	wsHost            string
	wsConn            *websocket.Conn
	client            *http.Client
	symbolPairNameMap map[string]entity.PairName
}

func NewHuobiService(apiHost, wsHost string) (*HuobiService, error) {
	s := &HuobiService{
		apiHost:           apiHost,
		wsHost:            wsHost,
		client:            &http.Client{Transport: &transport{}},
		symbolPairNameMap: make(map[string]entity.PairName),
	}
	symbols, err := s.getSymbols()
	if err != nil {
		return nil, err
	}
	for _, symbol := range symbols {
		s.symbolPairNameMap[symbol.Symbol] = symbol.PairName()
	}
	return s, nil
}

func (s *HuobiService) WsReceiveTicker() (<-chan entity.Ticker, error) {
	var (
		err      error
		tickerCh = make(chan entity.Ticker, 10)
	)
	s.wsConn, _, err = websocket.DefaultDialer.Dial(s.wsHost, nil)
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			v := &struct {
				Ch     string `json:"ch"`
				Ts     int64  `json:"ts"`
				Ping   int64  `json:"ping"`
				ErrMsg string `json:"err-msg"`
				Tick   struct {
					Close float64 `json:"close"`
				} `json:"tick"`
			}{}
			_, p, err := s.wsConn.ReadMessage()
			if err != nil {
				panic(err)
			}
			buf, err := gunzip(bytes.NewReader(p))
			if err != nil {
				log.Printf("[ERROR] gunzip message error: %s", err)
				continue
			}
			log.Printf("[INFO] data: %s", buf.String())
			err = json.NewDecoder(buf).Decode(v)
			if err != nil {
				log.Printf("[ERROR] json decode message error: %s", err)
				continue
			}
			if v.Ping > 0 {
				err = s.wsPong(v.Ping)
				if err != nil {
					log.Printf("[ERROR] ws send ping error: %s", err)
				}
				continue
			}
			if v.ErrMsg != "" {
				log.Printf("[ERROR] ws receive error message: %s", v.ErrMsg)
				continue
			}
			var pairName entity.PairName
			idx := strings.IndexRune(v.Ch, '.')
			if idx == -1 {
				continue
			}
			v.Ch = v.Ch[idx+1:]
			idx = strings.IndexRune(v.Ch, '.')
			if idx == -1 {
				continue
			}
			v.Ch = v.Ch[:idx]
			pairName, ok := s.symbolPairNameMap[v.Ch]
			if !ok {
				continue
			}
			tickerCh <- entity.Ticker{
				Vendor:   vendor,
				PairName: pairName,
				Last:     v.Tick.Close,
			}
		}
	}()

	for symbol := range s.symbolPairNameMap {
		err = s.wsRequestMarket(symbol)
		if err != nil {
			return nil, err
		}
		err = s.wsSubscribeMarket(symbol)
		if err != nil {
			return nil, err
		}
	}
	return tickerCh, nil
}

func (s *HuobiService) wsRequestMarket(market string) error {
	return s.wsConn.WriteJSON(map[string]string{
		"req": fmt.Sprintf("market.%s.detail", market),
	})
}

func (s *HuobiService) wsSubscribeMarket(market string) error {
	return s.wsConn.WriteJSON(map[string]string{
		"sub": fmt.Sprintf("market.%s.detail", market),
	})
}

func (s *HuobiService) wsPong(seq int64) error {
	return s.wsConn.WriteJSON(map[string]int64{"pong": seq})
}

func (s *HuobiService) getSymbols() ([]symbol, error) {
	reqURL := fmt.Sprintf("%s/v1/settings/symbols", s.apiHost)
	resp, err := s.client.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	v := &struct {
		Status string   `json:"status"`
		Data   []symbol `json:"data"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(v)
	if err != nil {
		return nil, err
	}
	if v.Status != "ok" {
		return nil, errors.New("status is not ok")
	}
	return v.Data, nil
}
