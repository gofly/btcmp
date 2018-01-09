package compare

import (
	"github.com/gofly/btcmp/entity"
)

func typeVendorsKey() string {
	return "compare:vendors"
}

func (s *CompareService) GetVendors() ([]entity.Vendor, error) {
	key := typeVendorsKey()
	return s.redisCli.LRange(key, 0, 1000).Result()
}

func (s *CompareService) SaveVendors(vendors []entity.Vendor) error {
	vendorSlice := make([]interface{}, len(vendors))
	for i, vendor := range vendors {
		vendorSlice[i] = vendor
	}
	key := typeVendorsKey()
	err := s.redisCli.LTrim(key, 0, -1).Err()
	if err != nil {
		return err
	}
	return s.redisCli.LPush(key, vendorSlice...).Err()
}
