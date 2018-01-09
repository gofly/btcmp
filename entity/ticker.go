package entity

type Ticker struct {
	Vendor   Vendor
	PairName PairName
	Last     float64
}

type Tickers []Ticker

type Vendor = string

type VendorTickerMap map[Vendor]Ticker
type PairVendorTickerMap map[PairName]VendorTickerMap
type PairTickerMap map[PairName]Ticker

func (ts Tickers) PairTypeTickersMap() map[PairType]Tickers {
	ptTsMap := make(map[PairType]Tickers)
	for _, ticker := range ts {
		pairType := ticker.PairName.PairType()
		if pairType == "" {
			continue
		}
		ptTsMap[pairType] = append(ptTsMap[pairType], ticker)
	}
	return ptTsMap
}

func (ts Tickers) PairNames() []PairName {
	pnMap := make(map[PairName]struct{})
	pairNames := make([]PairName, 0)
	for _, t := range ts {
		if _, ok := pnMap[t.PairName]; !ok {
			pnMap[t.PairName] = struct{}{}
			pairNames = append(pairNames, t.PairName)
		}
	}
	return pairNames
}

func (ts Tickers) PairVendorTickerMap() map[PairName]VendorTickerMap {
	pnvtMap := make(map[PairName]VendorTickerMap)
	for _, pairName := range ts.PairNames() {
		pnvtMap[pairName] = make(VendorTickerMap)
	}
	for _, t := range ts {
		pnvtMap[t.PairName][t.Vendor] = t
	}
	return pnvtMap
}
