package action

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gofly/btcmp/entity"
)

func (a *ReceiverAction) WsReceiveDataHandler(w http.ResponseWriter, r *http.Request) {
	whom := r.Header.Get("X-Whom")
	log.Printf("[INFO][WsReceiveDataHandler] %s connected", whom)
	ws, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ERROR][WsReceiveDataHandler] %s upgrade error: %s", whom, err)
		return
	}
	defer ws.Close()

	log.Printf("[INFO][WsReceiveDataHandler] %s upgraded", whom)
	for {
		var tickers entity.Tickers
		err := ws.ReadJSON(&tickers)
		if err != nil {
			log.Printf("[WARN][WsReceiveDataHandler] %s ReadJSON error: %s", whom, err)
			break
		}
		log.Printf("[INFO][WsReceiveDataHandler] receive tickers from %s, len: %d, data: %+v",
			whom, len(tickers), tickers)

		err = a.cmpSrv.SaveTickers(tickers)
		if err != nil {
			log.Printf("[ERROR][WsReceiveDataHandler] %s SaveTickers error: %s", whom, err)
		}
		mtTickersMap := tickers.PairTypeTickersMap()
		for pairType, _tickers := range mtTickersMap {
			data, _ := json.Marshal(_tickers.PairVendorTickerMap())
			err = a.redisCli.Publish(boardcastPairsKey(pairType), data).Err()
			if err != nil {
				log.Printf("[ERROR][WsReceiveDataHandler] %s Publish error: %s", whom, err)
			}
		}
	}
}
