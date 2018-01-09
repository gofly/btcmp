package compare

import (
	"encoding/json"
	"fmt"

	"github.com/gofly/btcmp/entity"
)

func settingsKey() string {
	return fmt.Sprintf("compare:settings")
}

func (s *CompareService) GetUserSettings(userName string) (*entity.Settings, error) {
	key := settingsKey()
	data, err := s.redisCli.HGet(key, userName).Bytes()
	if err != nil {
		return nil, err
	}
	settings := &entity.Settings{}
	err = json.Unmarshal(data, &settings)
	return settings, err
}

func (s *CompareService) SaveUserSettings(userName string, settings *entity.Settings) error {
	key := settingsKey()
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return s.redisCli.HSet(key, userName, data).Err()
}

func (s *CompareService) GetUserAttentedPairNames(userName string) (entity.PairNames, error) {
	key := settingsKey()
	data, err := s.redisCli.HGet(key, userName).Bytes()
	if err != nil {
		return nil, err
	}
	settings := &entity.Settings{}
	err = json.Unmarshal(data, settings)
	if err != nil {
		return nil, err
	}
	pairNames := make(entity.PairNames, 0)
	for _, threshold := range settings.Thresholds {
		pairNames = append(pairNames, threshold.PairName)
	}
	return pairNames, nil
}

func (s *CompareService) GetAllAttentedPairNames() (entity.PairNames, error) {
	key := settingsKey()
	data, err := s.redisCli.HGetAll(key).Result()
	if err != nil {
		return nil, err
	}
	pairMap := make(map[entity.PairName]struct{})
	pairNames := make(entity.PairNames, 0)
	for _, byts := range data {
		settings := &entity.Settings{}
		err := json.Unmarshal([]byte(byts), settings)
		if err != nil {
			continue
		}
		for _, threshold := range settings.Thresholds {
			if _, ok := pairMap[threshold.PairName]; !ok {
				pairNames = append(pairNames, threshold.PairName)
			}
		}
	}
	return pairNames, nil
}
