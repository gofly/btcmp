package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"strings"
	"time"

	"github.com/gofly/btcmp/entity"

	"github.com/gorilla/websocket"
)

const (
	vendor = "Gateio"
)

var (
	ErrUnexpectedMessage = errors.New("unexpected message")
)

type GateIOService struct {
	dataHost string
	wsHost   string
	client   *http.Client
}

func NewGateIOService(dataHost, wsHost string) *GateIOService {
	jar, _ := cookiejar.New(nil)
	return &GateIOService{
		dataHost: dataHost,
		wsHost:   wsHost,
		client: &http.Client{
			Jar: jar,
		},
	}
}

func nowTSToStr() string {
	return strconv.FormatInt(time.Now().Unix(), 36)
}

func (s *GateIOService) getWSSid() (string, error) {
	reqURL := fmt.Sprintf("https://%s/socket.io/?EIO=3&transport=polling&t=%s", s.wsHost, nowTSToStr())
	resp, err := s.client.Get(reqURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get ws sid return status: %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	idx1 := bytes.IndexByte(body, '{')
	idx2 := bytes.LastIndexByte(body, '}')
	body = body[idx1 : idx2+1]
	rtn := &struct {
		Sid string `json:"sid"`
	}{}
	err = json.Unmarshal(body, rtn)
	if err != nil {
		return "", err
	}
	return rtn.Sid, nil
}

func (s *GateIOService) joinWSCode(sid, pairCode string) error {
	reqURL := fmt.Sprintf("https://%s/socket.io/?EIO=3&transport=polling&t=%s&sid=%s", s.wsHost, nowTSToStr(), sid)
	log.Println(reqURL)
	joinMsg := fmt.Sprintf(`42["join","gateio_%s"]`, pairCode)
	resp, err := s.client.Post(reqURL,
		"text/plain;charset=UTF-8",
		strings.NewReader(fmt.Sprintf(`%d:%s`, len(joinMsg), joinMsg)),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("join websocket return status: %d", resp.StatusCode)
	}
	return nil
}

func (s *GateIOService) dialWS(sid string) (*websocket.Conn, error) {
	wsURL := fmt.Sprintf("wss://%s/socket.io/?EIO=3&transport=websocket&sid=%s", s.wsHost, sid)
	wsConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	return wsConn, err
}

func (s *GateIOService) wsPing(wsConn *websocket.Conn) error {
	return wsConn.WriteMessage(websocket.TextMessage, []byte("2"))
}

func (s *GateIOService) wsHandshake(wsConn *websocket.Conn) error {
	err := wsConn.WriteMessage(websocket.TextMessage, []byte("2probe"))
	if err != nil {
		return err
	}
	_, msg, err := wsConn.ReadMessage()
	if err != nil {
		return err
	}
	if string(msg) == "3probe" {
		return wsConn.WriteMessage(websocket.TextMessage, []byte("5"))
	}
	return fmt.Errorf("handshake error, msg is %s", msg)
}

func (s *GateIOService) parseMessage(pairName entity.PairName, message []byte) (*entity.Ticker, error) {
	data := &struct {
		Data struct {
			Result   bool    `json:"result"`
			AskRate0 float64 `json:"ask_rate0"`
			BidRate0 float64 `json:"bid_rate0"`
		} `json:"data"`
	}{}
	err := json.Unmarshal(message, data)
	if err != nil {
		return nil, err
	}
	if !data.Data.Result {
		return nil, errors.New("result is not true")
	}
	return &entity.Ticker{
		Vendor:   vendor,
		PairName: pairName,
		Last:     (data.Data.AskRate0 + data.Data.BidRate0) / 2,
	}, nil
}

func (s *GateIOService) ReceiveData(pairName entity.PairName, pairCode string, tickerChan chan<- entity.Ticker) error {
	sid, err := s.getWSSid()
	if err != nil {
		return err
	}
	err = s.joinWSCode(sid, pairCode)
	if err != nil {
		return err
	}
	wsConn, err := s.dialWS(sid)
	if err != nil {
		return err
	}
	s.wsHandshake(wsConn)
	if err != nil {
		return err
	}

	go func() {
		pingTicker := time.Tick(time.Second * 25)
	Loop:
		for {
			select {
			case <-pingTicker:
				err := s.wsPing(wsConn)
				if err != nil {
					log.Printf("[ERROR] pair %s ws ping error: %s", pairName, err)
					time.Sleep(time.Second * 5)
					wsConn, err = s.dialWS(sid)
					if err != nil {
						log.Printf("[ERROR] pair %s ws dial in ping error: %s", pairName, err)
						continue Loop
					}
				}
			default:
				_, p, err := wsConn.ReadMessage()
				if err != nil {
					log.Printf("[ERROR] pair %s ws read message error: %s", pairName, err)
					time.Sleep(time.Second * 5)
					wsConn, err = s.dialWS(sid)
					if err != nil {
						log.Printf("[ERROR] pair %s ws dial in read message error: %s", pairName, err)
						continue Loop
					}
				}
				log.Printf("[INFO] data: ", string(p))
				if len(p) > 18 {
					p = p[16 : len(p)-1]
					ticker, err := s.parseMessage(pairName, p)
					if err != nil {
						log.Printf("[ERROR]parseMessage error: %s", err)
						continue Loop
					}
					tickerChan <- *ticker
				}
			}
		}
	}()
	return nil
}

func (s *GateIOService) GetPairs() (map[entity.PairName]string, error) {
	res, err := s.client.Get(fmt.Sprintf("http://%s/api2/1/pairs", s.dataHost))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("get ticker %s from bter return status: %d", res.StatusCode)
		return nil, err
	}
	pairs := make([]string, 0)
	err = json.NewDecoder(res.Body).Decode(&pairs)
	if err != nil {
		return nil, err
	}

	pairCodeMap := make(map[entity.PairName]string)
	for _, pair := range pairs {
		parts := strings.Split(pair, "_")
		if len(parts) != 2 {
			continue
		}
		pairName := entity.PairName(strings.ToUpper(parts[1] + "-" + parts[0]))
		pairCodeMap[pairName] = pair
	}
	return pairCodeMap, nil
}
