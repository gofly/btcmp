package action

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gofly/btcmp/entity"
)

const userName = "huying"

func apiSuccessResponse(w http.ResponseWriter) {
	w.Write([]byte(`{"sucess": true}`))
}

func boardcastPairsKey(pairType entity.PairType) string {
	return fmt.Sprintf("boardcast:pairs:%s", pairType)
}

func boardcastSettingsUpdateKey(userName string) string {
	return fmt.Sprintf("boardcast:settings.update:%s", userName)
}

func pairTypeInURI(reqURI string) entity.PairType {
	index := strings.LastIndex(reqURI, "/")
	return entity.PairType(strings.ToUpper(reqURI[index+1:]))
}

func serviceName(reqURI string) string {
	index := strings.LastIndex(reqURI, "/")
	return reqURI[index+1:]
}
