package compare

import (
	"encoding/json"
	"fmt"

	"github.com/gofly/btcmp/entity"
)

func vendorTickersKey(pairName entity.PairName) string {
	return fmt.Sprintf("compare:tickers:%s", pairName)
}

func (s *CompareService) GetVendorTickerMapByPairName(pairName entity.PairName) (entity.VendorTickerMap, error) {
	tickerMap := make(entity.VendorTickerMap)
	key := vendorTickersKey(pairName)
	results, err := s.redisCli.HGetAll(key).Result()
	if err != nil {
		return nil, err
	}
	for vendor, result := range results {
		ticker := entity.Ticker{}
		err := json.Unmarshal([]byte(result), &ticker)
		if err != nil {
			return nil, err
		}
		tickerMap[vendor] = ticker
	}
	return tickerMap, nil
}

func (s *CompareService) SaveTickers(tickers entity.Tickers) error {
	for _, ticker := range tickers {
		key := vendorTickersKey(ticker.PairName)
		data, err := json.Marshal(ticker)
		if err != nil {
			return err
		}
		err = s.redisCli.HSet(key, ticker.Vendor, data).Err()
		if err != nil {
			return err
		}
	}
	return nil
}
