package entity

type Threshold struct {
	PairName PairName
	Value    float64
}

type Thresholds []Threshold

func (ts Thresholds) GroupByPairType() map[PairType]Thresholds {
	tptMap := make(map[PairType]Thresholds)
	for _, t := range ts {
		pairType := t.PairName.PairType()
		if pairType == "" {
			continue
		}
		tptMap[pairType] = append(tptMap[pairType], t)
	}
	return tptMap
}
