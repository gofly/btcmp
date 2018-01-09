package action

import (
	"fmt"
	"log"
	"net/http"
)

func (a *WebAction) PageCompare(w http.ResponseWriter, r *http.Request) {
	pairType := pairTypeInURI(r.RequestURI)
	pairTypes, err := a.cmpSrv.GetPairTypes()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "GetPairTypes() error: %s", err)
		log.Printf("[ERROR] GetPairTypes() error: %s", err)
		return
	}
	err = a.tmpl.ExecuteTemplate(w, "compare.html", map[string]interface{}{
		"PairType":   pairType,
		"PairTypes":  pairTypes,
		"WsDataHost": r.Host,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "render template error: %s", err)
		log.Printf("[ERROR] render template error: %s", err)
	}
}

func (a *WebAction) PageSettings(w http.ResponseWriter, r *http.Request) {
	pairTypes, err := a.cmpSrv.GetPairTypes()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "GetPairTypes() error: %s", err)
		return
	}
	settings, err := a.cmpSrv.GetUserSettings(userName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "GetUserThresholds(%s) error: %s", userName, err)
		log.Printf("[ERROR] GetUserThresholds(%s) error: %s", userName, err)
		return
	}
	processes, err := a.superCli.GetAllProcessInfo()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "GetAllProcessInfo() error: %s", err)
		log.Printf("[ERROR] GetAllProcessInfo() error: %s", err)
		return
	}
	err = a.tmpl.ExecuteTemplate(w, "settings.html", map[string]interface{}{
		"PairTypes":    pairTypes,
		"MusicNotify":  settings.MusicNotify,
		"Thresholds":   settings.Thresholds,
		"ProcessInfos": processes,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "ExecuteTemplate error: %s", err)
		log.Printf("[ERROR] ExecuteTemplate error: %s", err)
		return
	}
}
