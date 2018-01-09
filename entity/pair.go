package entity

import (
	"strings"
)

type PairType string

func (t PairType) ToLower() string {
	return strings.ToLower(string(t))
}

func (t PairType) String() string {
	return string(t)
}

type PairName string

func (n PairName) PairType() PairType {
	index := strings.IndexRune(string(n), '-')
	if index != -1 {
		return PairType(n[:index])
	}
	return ""
}

func (n PairName) BaseName() string {
	index := strings.IndexRune(n.String(), '-')
	if index != -1 {
		return n[index+1:].String()
	}
	return ""
}

func (n PairName) String() string {
	return string(n)
}

type PairNames []PairName

func (mns PairNames) Types() []PairType {
	pairTypes := make([]PairType, 0)
	pairTypeMap := make(map[PairType]struct{})
	for _, pairName := range mns {
		pairType := pairName.PairType()
		if pairType == "" {
			continue
		}
		if _, ok := pairTypeMap[pairType]; !ok {
			pairTypes = append(pairTypes, pairType)
		}
	}
	return pairTypes
}

func (mns PairNames) GroupByPairType() map[PairType]PairNames {
	mtmnMap := make(map[PairType]PairNames)
	for _, pairName := range mns {
		pairType := pairName.PairType()
		if pairType == "" {
			continue
		}
		mtmnMap[pairType] = append(mtmnMap[pairType], pairName)
	}
	return mtmnMap
}

type PairNameRewriteMap map[PairName]PairName
