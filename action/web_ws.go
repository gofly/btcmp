package action

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gofly/btcmp/entity"
)

func (a *WebAction) filterPairData(userName string, pvtMap entity.PairVendorTickerMap) (entity.PairVendorTickerMap, error) {
	pairNames, err := a.cmpSrv.GetUserAttentedPairNames(userName)
	if err != nil {
		return nil, err
	}
	rewriteMap, err := a.cmpSrv.GetPairNameRewrite()
	if err != nil {
		return nil, err
	}
	pairNameMap := make(map[entity.PairName]struct{})
	for _, pairName := range pairNames {
		pairNameMap[pairName] = struct{}{}
	}
	for pairName, vtMap := range pvtMap {
		if pn, ok := rewriteMap[pairName]; ok {
			if _, ok := pvtMap[pn]; !ok {
				pvtMap[pn] = make(entity.VendorTickerMap)
			}
			for vendor, tMap := range vtMap {
				pvtMap[pn][vendor] = tMap
			}
			delete(pvtMap, pairName)
			continue
		}
		if _, ok := pairNameMap[pairName]; !ok {
			delete(pvtMap, pairName)
		}
	}
	return pvtMap, nil
}

func (a *WebAction) WsPairData(w http.ResponseWriter, r *http.Request) {
	pairType := pairTypeInURI(r.RequestURI)
	log.Printf("[INFO][WsPairData] connected (user: %s, pairtype: %s)", userName, pairType)
	ws, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ERROR][WsPairData] upgrade (user: %s, pairtype: %s) error: %s", userName, pairType)
		fmt.Fprintf(w, "ws upgrade error: %s", err)
		return
	}
	defer ws.Close()
	log.Printf("[INFO][WsPairData] upgrade (user: %s, pairtype: %s)", userName, pairType)

	ctx, cancelFunc := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	dataCh := make(chan interface{})
	pongCh := make(chan struct{})

	go func() {
		for {
			msg := &struct{ Method string }{}
			err := ws.ReadJSON(msg)
			if err != nil {
				cancelFunc()
				return
			}
			if msg.Method == "Pong" {
				pongCh <- struct{}{}
			}
		}
	}()

	go func() {
		pingTicker := time.NewTicker(time.Second * 5)
		waitPong := false
		for {
			select {
			case <-ctx.Done():
				pingTicker.Stop()
				wg.Wait()
				close(dataCh)
				return
			case <-pingTicker.C:
				if waitPong {
					cancelFunc()
				} else {
					wg.Add(1)
					dataCh <- map[string]string{"Method": "Ping"}
					waitPong = true
					log.Printf("[INFO][WsPairData] ping (user: %s, pairtype: %s)", userName, pairType)
				}
			case <-pongCh:
				waitPong = false
				log.Printf("[INFO][WsPairData] receive pong (user: %s, pairtype: %s)", userName, pairType)
			}
		}
	}()

	go func() {
		pairsChannel := boardcastPairsKey(pairType)
		settingsChannel := boardcastSettingsUpdateKey(userName)
		pubSub := a.redisCli.Subscribe(pairsChannel, settingsChannel)
		go func() {
			<-ctx.Done()
			pubSub.Close()
		}()
		for {
			message, err := pubSub.ReceiveMessage()
			if err != nil {
				break
			}
			switch message.Channel {
			case settingsChannel:
				wg.Add(1)
				dataCh <- map[string]string{"Method": "Thresholds.Reload"}
			case pairsChannel:
				pvtMap := make(entity.PairVendorTickerMap)
				err = json.Unmarshal([]byte(message.Payload), &pvtMap)
				if err != nil {
					log.Printf("[WARN][WsPairData] unmarshal payload json error: %s", err)
					continue
				}
				pvtMap, err = a.filterPairData(userName, pvtMap)
				if err != nil {
					log.Printf("[ERROR][WsPairData] GetUserAttentedPairNames error: %s", err)
					continue
				}
				if len(pvtMap) == 0 {
					continue
				}
				wg.Add(1)
				dataCh <- map[string]interface{}{
					"Method": "Tickers.Update",
					"Data":   pvtMap,
				}
			}
		}
		cancelFunc()
	}()

	for data := range dataCh {
		wg.Done()
		ws.SetWriteDeadline(time.Now().Add(time.Second))
		err = ws.WriteJSON(data)
		if err != nil {
			cancelFunc()
		}
	}
	log.Printf("[INFO][WsPairData] user: %s, pairtype: %s exit", userName, pairType)
}
