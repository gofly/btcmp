package action

import (
	"html/template"
	"net/http"
	"time"

	supervisord "github.com/abrander/go-supervisord"
	"github.com/go-redis/redis"
	"github.com/gofly/btcmp/service/compare"
	"github.com/gorilla/websocket"
)

type WebAction struct {
	redisCli   redis.UniversalClient
	upgrader   *websocket.Upgrader
	cmpSrv     *compare.CompareService
	wsPongWait time.Duration
	tmpl       *template.Template
	superCli   *supervisord.Client
}

func NewWebAction(redisCli redis.UniversalClient, cmpSrv *compare.CompareService, wsPongWait time.Duration, tmpl *template.Template, superCli *supervisord.Client) *WebAction {
	return &WebAction{
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
		tmpl:       tmpl,
		superCli:   superCli,
	}
}
