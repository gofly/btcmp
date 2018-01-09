package main

import (
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	supervisord "github.com/abrander/go-supervisord"
	"github.com/go-redis/redis"
	"github.com/gofly/btcmp/action"
	"github.com/gofly/btcmp/config"
	"github.com/gofly/btcmp/service/compare"
	yaml "gopkg.in/yaml.v2"
)

var (
	redisCli   redis.UniversalClient
	webAction  *action.WebAction
	listenAddr string
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
	confPath := flag.String("config", "web.yml", "config file")
	flag.Parse()
	conf, err := loadConfig(*confPath)
	if err != nil {
		panic(err)
	}
	listenAddr = conf.Listen
	redisCli = redis.NewUniversalClient(&conf.Redis)
	cmpSrv := compare.NewCompareService(redisCli)
	log.Printf("conf: %v\n", *conf)

	tmpl, err := template.ParseGlob("views/*.html")
	if err != nil {
		panic(err)
	}
	superCli, err := supervisord.NewClient(conf.SupervisorRPC)
	if err != nil {
		panic(err)
	}
	webAction = action.NewWebAction(redisCli, cmpSrv, time.Second*5, tmpl, superCli)
}

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/compare/", webAction.PageCompare)
	http.HandleFunc("/settings", webAction.PageSettings)
	http.HandleFunc("/api/pairs/", webAction.ApiGetPairsData)
	http.HandleFunc("/api/pairs/rewrite", webAction.ApiPairsRewrite)
	http.HandleFunc("/api/settings", webAction.ApiSaveSettings)
	http.HandleFunc("/api/settings/", webAction.ApiGetSettings)
	http.HandleFunc("/api/restart", webAction.ApiRestart)
	http.HandleFunc("/ws/pairs/", webAction.WsPairData)

	http.ListenAndServe(listenAddr, nil)
}
