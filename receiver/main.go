package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/go-redis/redis"
	"github.com/gofly/btcmp/action"
	"github.com/gofly/btcmp/config"
	"github.com/gofly/btcmp/service/compare"
	yaml "gopkg.in/yaml.v2"
)

var (
	redisCli       redis.UniversalClient
	receiverAction *action.ReceiverAction
	listenAddr     string
)

func loadConfig(confPath string) (*config.Config, error) {
	content, err := ioutil.ReadFile(confPath)
	if err != nil {
		return nil, err
	}
	conf := &config.Config{}
	err = yaml.UnmarshalStrict(content, conf)
	return conf, err
}

func init() {
	confPath := flag.String("config", "receiver.yml", "config file")
	flag.Parse()
	conf, err := loadConfig(*confPath)
	if err != nil {
		panic(err)
	}
	listenAddr = conf.Listen
	redisCli = redis.NewUniversalClient(&conf.Redis)
	cmpSrv := compare.NewCompareService(redisCli)
	log.Printf("conf: %v\n", *conf)

	receiverAction = action.NewReceiverAction(redisCli, cmpSrv, time.Second*5)
}

func main() {
	http.HandleFunc("/api/pair_types", receiverAction.ApiSavePairTypes)
	http.HandleFunc("/api/attentions/pairs", receiverAction.ApiGetAttentedPairNames)
	http.HandleFunc("/ws/pairs", receiverAction.WsReceiveDataHandler)

	http.ListenAndServe(listenAddr, nil)
}
