package compare

import (
	"encoding/json"

	"github.com/gofly/btcmp/entity"
)

func usersKey() string {
	return "compare:users"
}

func (s *CompareService) GetAllUsers() ([]entity.User, error) {
	data, err := s.redisCli.HGetAll(usersKey()).Result()
	if err != nil {
		return nil, err
	}
	users := make([]entity.User, 0)
	for _, userData := range data {
		user := entity.User{}
		err = json.Unmarshal([]byte(userData), &user)
		if err != nil {
			continue
		}
		users = append(users, user)
	}
	return users, nil
}

func (s *CompareService) GetUser(userName string) (*entity.User, error) {
	data, err := s.redisCli.HGet(usersKey(), userName).Bytes()
	if err != nil {
		return nil, err
	}
	user := &entity.User{}
	err = json.Unmarshal(data, user)
	return user, err
}

func (s *CompareService) SaveUser(user *entity.User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return s.redisCli.HSet(usersKey(), user.UserName, data).Err()
}
