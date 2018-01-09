package action

import (
	"net/http"
	"time"

	"github.com/go-redis/redis"
	"github.com/gofly/btcmp/service/compare"
	"github.com/gorilla/websocket"
)

type ReceiverAction struct {
	redisCli   redis.UniversalClient
	upgrader   *websocket.Upgrader
	cmpSrv     *compare.CompareService
	wsPongWait time.Duration
}

func NewReceiverAction(redisCli redis.UniversalClient, cmpSrv *compare.CompareService, wsPongWait time.Duration) *ReceiverAction {
	action := &ReceiverAction{
		redisCli: redisCli,
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  2 << 10,
			WriteBufferSize: 2 << 10,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		cmpSrv:     cmpSrv,
		wsPongWait: wsPongWait,
	}
	return action
}
