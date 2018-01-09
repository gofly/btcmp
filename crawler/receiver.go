package crawler

import (
	"encoding/json"
	"net/http"

	"github.com/gofly/btcmp/entity"
	"github.com/gorilla/websocket"
)

type ReceiverService struct {
	host   string
	vendor string
	ws     *websocket.Conn
}

func NewReceiverService(host, vendor string) *ReceiverService {
	return &ReceiverService{
		host:   host,
		vendor: vendor,
	}
}
func (s *ReceiverService) DialWs() (err error) {
	wsURL := "ws://" + s.host + "/ws/pairs"
	s.ws, _, err = websocket.DefaultDialer.Dial(wsURL, http.Header{
		"X-Whom": []string{s.vendor},
	})
	return
}

func (s *ReceiverService) CloseWs() error {
	return s.ws.Close()
}

func (s *ReceiverService) SendTickers(tickersCh <-chan entity.Tickers) <-chan error {
	errCh := make(chan error)
	go func() {
		for tickers := range tickersCh {
			err := s.ws.WriteJSON(tickers)
			if err != nil {
				errCh <- err
				break
			}
		}
	}()
	return errCh
}

func (s *ReceiverService) GetAllAttendedPairNames() (entity.PairNames, error) {
	resp, err := http.Get("http://" + s.host + "/api/attentions/pairs")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	pairNames := make(entity.PairNames, 0)
	err = json.NewDecoder(resp.Body).Decode(&pairNames)
	return pairNames, err
}
