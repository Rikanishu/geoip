package main

import (
	btree "github.com/Rikanishu/btree/ui32"
	"github.com/sirupsen/logrus"
	"testing"
)

var testStorage *CountryStorage

func init() {
	conf, err := ParseConfig("./config.yaml")
	if err != nil {
		panic(err)
	}
	testStorage = NewCountryStorage(conf, BuildCountryDataSource(conf))
	testStorage.lock.Lock()
	defer testStorage.lock.Unlock()

	treeLen := 0
	testStorage.tree.Ascend(func(i *btree.Item) bool {
		treeLen += 1
		if i.SubTree != nil {
			treeLen += i.SubTree.Len()
		}
		return true
	})
	logrus.Info(treeLen)
}

func BenchmarkFindAvail(b *testing.B) {
	ip, err := ipv4toUint32("153.98.72.15")
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		x := testStorage.FindCountry(ip)
		_ = x
	}
}
