package action

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gofly/btcmp/entity"
)

func (a *ReceiverAction) ApiGetAttentedPairNames(w http.ResponseWriter, r *http.Request) {
	pairNames, err := a.cmpSrv.GetAllAttentedPairNames()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "GetAllAttentedPairNames() error: %s", err)
		log.Printf("[ERROR] GetAllAttentedPairNames() error: %s", err)
		return
	}
	json.NewEncoder(w).Encode(pairNames)
}

func (a *ReceiverAction) ApiSavePairTypes(w http.ResponseWriter, r *http.Request) {
	pairTypes := make([]entity.PairType, 0)
	err := json.NewDecoder(r.Body).Decode(&pairTypes)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "json decode error: %s", err)
		log.Printf("[ERROR] json decode error: %s", err)
		return
	}
	err = a.cmpSrv.SavePairTypes(pairTypes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "SavePairTypes() error: %s", err)
		log.Printf("[ERROR] SavePairTypes(%+v) error: %s", pairTypes, err)
		return
	}
	apiSuccessResponse(w)
}
