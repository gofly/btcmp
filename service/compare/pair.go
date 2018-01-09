package compare

import (
	"encoding/json"
	"fmt"

	"github.com/gofly/btcmp/entity"
)

func typePairsKey() string {
	return fmt.Sprintf("compare:pairs")
}

func (s *CompareService) GetPairNamesByPairType(pairType entity.PairType) (entity.PairNames, error) {
	key := typePairsKey()
	data, err := s.redisCli.HGet(key, pairType.String()).Bytes()
	if err != nil {
		return nil, err
	}
	pairNames := make(entity.PairNames, 0)
	err = json.Unmarshal(data, &pairNames)
	return pairNames, err
}

func (s *CompareService) SavePairNames(pairNames entity.PairNames) error {
	key := typePairsKey()
	for mType, mNames := range pairNames.GroupByPairType() {
		data, err := json.Marshal(mNames)
		if err != nil {
			continue
		}
		keys, err := s.redisCli.HKeys(key).Result()
		if err != nil {
			return err
		}
		err = s.redisCli.HDel(key, keys...).Err()
		if err != nil {
			return err
		}
		err = s.redisCli.HSet(key, mType.String(), data).Err()
		if err != nil {
			return err
		}
	}
	return nil
}

func typePairTypesKey() string {
	return "compare:pair_types"
}

func (s *CompareService) GetPairTypes() ([]entity.PairType, error) {
	key := typePairTypesKey()
	strs, err := s.redisCli.LRange(key, 0, 1000).Result()
	if err != nil {
		return nil, err
	}
	pairTypes := make([]entity.PairType, len(strs))
	for i, str := range strs {
		pairTypes[i] = entity.PairType(str)
	}
	return pairTypes, nil
}

func (s *CompareService) SavePairTypes(pairTypes []entity.PairType) error {
	pairTypeSlice := make([]interface{}, len(pairTypes))
	for i, pairType := range pairTypes {
		pairTypeSlice[i] = pairType.String()
	}
	key := typePairTypesKey()
	err := s.redisCli.LTrim(key, 1, 0).Err()
	if err != nil {
		return err
	}
	return s.redisCli.RPush(key, pairTypeSlice...).Err()
}

func pairNameRewriteKey() string {
	return "compare:pairname:rewrite"
}

func (s *CompareService) SavePairNameRewrite(rewriteMap map[entity.PairName]entity.PairName) error {
	data := make(map[string]interface{})
	for k, v := range rewriteMap {
		data[k.String()] = v.String()
	}
	return s.redisCli.HMSet(pairNameRewriteKey(), data).Err()
}

func (s *CompareService) GetPairNameRewrite() (map[entity.PairName]entity.PairName, error) {
	data, err := s.redisCli.HGetAll(pairNameRewriteKey()).Result()
	if err != nil {
		return nil, err
	}
	rewriteMap := make(map[entity.PairName]entity.PairName)
	for k, v := range data {
		rewriteMap[entity.PairName(k)] = entity.PairName(v)
	}
	return rewriteMap, nil
}
