package compare

import (
	"github.com/go-redis/redis"
)

type CompareService struct {
	redisCli redis.UniversalClient
}

func NewCompareService(redisCli redis.UniversalClient) *CompareService {
	return &CompareService{
		redisCli: redisCli,
	}
}
