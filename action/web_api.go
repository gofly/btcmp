package action

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gofly/btcmp/entity"
)

func (a *WebAction) getPairVendorData(pairNames entity.PairNames) ([]entity.PairVendorTickerMap, error) {
	pvtMapMap := make(map[entity.PairName]entity.VendorTickerMap)

	rewriteMap, err := a.cmpSrv.GetPairNameRewrite()
	if err != nil {
		return nil, err
	}
	reverseMap := make(entity.PairNameRewriteMap)
	for k, v := range rewriteMap {
		reverseMap[v] = k
	}
	for _, pairName := range pairNames {
		if pn, ok := reverseMap[pairName]; ok {
			pairNames = append(pairNames, pn)
		}
	}
	for _, pairName := range pairNames {
		vtMap, err := a.cmpSrv.GetVendorTickerMapByPairName(pairName)
		if err != nil {
			return nil, err
		}
		if pn, ok := rewriteMap[pairName]; ok {
			pairName = pn
		}
		if _, ok := pvtMapMap[pairName]; !ok {
			pvtMapMap[pairName] = make(entity.VendorTickerMap)
		}
		for vendor, tMap := range vtMap {
			pvtMapMap[pairName][vendor] = tMap
		}
	}
	pvtMaps := make([]entity.PairVendorTickerMap, 0)
	for _, pairName := range pairNames {
		if _, ok := rewriteMap[pairName]; ok {
			continue
		}
		pvtMaps = append(pvtMaps, entity.PairVendorTickerMap{pairName: pvtMapMap[pairName]})
	}
	return pvtMaps, nil
}

func (a *WebAction) ApiGetPairsData(w http.ResponseWriter, r *http.Request) {
	pairType := pairTypeInURI(r.RequestURI)
	pairNames, err := a.cmpSrv.GetUserAttentedPairNames(userName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "GetUserAttentedPairNames(%s) error: %s", userName, err)
		log.Printf("[ERROR] GetUserAttentedPairNames(%s) error: %s", userName, err)
		return
	}
	pairTypeNamesMap := pairNames.GroupByPairType()
	data, err := a.getPairVendorData(pairTypeNamesMap[pairType])
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Action.getPairVendorData(%s) error: %s", pairType, err)
		log.Printf("[ERROR] Action.getPairVendorData(%s) error: %s", pairType, err)
		return
	}
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	json.NewEncoder(w).Encode(data)
}

func (a *WebAction) ApiSavePairsRewrite(w http.ResponseWriter, r *http.Request) {
	rewriteMap := make(map[entity.PairName]entity.PairName)
	err := json.NewDecoder(r.Body).Decode(&rewriteMap)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "decode body error: %s", err)
		return
	}
	err = a.cmpSrv.SavePairNameRewrite(rewriteMap)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "SavePairNameRewrite(%s) error: %s", rewriteMap, err)
		log.Printf("[ERROR] SavePairNameRewrite(%s) error: %s", rewriteMap, err)
		return
	}
	apiSuccessResponse(w)
}

func (a *WebAction) ApiGetPairsRewrite(w http.ResponseWriter, r *http.Request) {
	rewriteMap, err := a.cmpSrv.GetPairNameRewrite()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "GetPairNameRewrite() error: %s", err)
		log.Printf("[ERROR] GetPairNameRewrite() error: %s", err)
		return
	}
	json.NewEncoder(w).Encode(rewriteMap)
}
func (a *WebAction) ApiSaveSettings(w http.ResponseWriter, r *http.Request) {
	settings := &entity.Settings{}
	err := json.NewDecoder(r.Body).Decode(&settings)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "[ERROR] decode user(%s)'s settings error: %s", userName, err)
		log.Printf("[ERROR] decode user(%s)'s settings error: %s", userName, err)
		return
	}
	log.Printf("[INFO] user(%s)'s settings: %+v", settings)
	err = a.cmpSrv.SaveUserSettings(userName, settings)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "SaveUesrSettings(%s, %v) error: %s", userName, settings, err)
		log.Printf("[ERROR] SaveUesrSettings(%s, %v) error: %s", userName, settings, err)
		return
	}
	err = a.redisCli.Publish(boardcastSettingsUpdateKey(userName), nil).Err()
	if err != nil {
		log.Printf("[ERROR] boardcast thresholds update for user %s error: %s", userName, err)
	}
	apiSuccessResponse(w)
}

func (a *WebAction) ApiPairsRewrite(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		a.ApiSavePairsRewrite(w, r)
	case http.MethodGet:
		a.ApiGetPairsRewrite(w, r)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (a *WebAction) ApiGetSettings(w http.ResponseWriter, r *http.Request) {
	pairType := pairTypeInURI(r.RequestURI)
	settings, err := a.cmpSrv.GetUserSettings(userName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "GetUserSettings(%s) error: %s", userName, err)
		log.Printf("[ERROR] GetUserSettings(%s) error: %s", userName, err)
		return
	}
	tMap := settings.Thresholds.GroupByPairType()
	if thresholds, ok := tMap[pairType]; ok {
		json.NewEncoder(w).Encode(entity.Settings{
			MusicNotify: settings.MusicNotify,
			Thresholds:  thresholds,
		})
		return
	}
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "pair type %s not found", pairType)
}

func (a *WebAction) ApiRestart(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	process := r.FormValue("process")
	err := a.superCli.StopProcess(process, true)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "stop process error: %s", err)
		log.Printf("[ERROR] StopProcess(%s) error: %s", process, err)
		return
	}
	err = a.superCli.StartProcess(process, true)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "start process error: %s", err)
		log.Printf("[ERROR] StartProcess(%s) error: %s", process, err)
		return
	}
	info, err := a.superCli.GetProcessInfo(process)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "get process info error: %s", err)
		log.Printf("[ERROR] GetProcessInfo(%s) error: %s", process, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(info)
}
