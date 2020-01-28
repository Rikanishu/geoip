package main

import (
	"sync"
	"time"

	btree "github.com/Rikanishu/btree/ui32"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type CountryStorage struct {
	config     *Config
	dataSource CountryDataSource
	tree       *btree.BTree
	lock       sync.RWMutex
}

func NewCountryStorage(config *Config, dataSource CountryDataSource) *CountryStorage {
	s := &CountryStorage{
		config:     config,
		dataSource: dataSource,
	}

	s.lock.Lock()
	go func() {
		defer s.lock.Unlock()

		err := s.build()
		if err != nil {
			logrus.Fatal(errors.Wrap(err, "unable to initialize storage"))
		}
	}()

	return s
}

func (s *CountryStorage) Refresh() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.build()
}

func (s *CountryStorage) FindCountry(ip uint32) *Country {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.tree == nil {
		return nil
	}

	out := make([]*Country, 0, 1)
	s.tree.DescendLessOrEqual(&btree.Item{
		Key: ip,
	}, func(item *btree.Item) bool {
		item.SubTree.AscendGreaterOrEqual(&btree.Item{
			Key: ip,
		}, func(item *btree.Item) bool {
			country := item.Payload.(*Country)
			out = append(out, country)
			return true
		})
		return true
	})

	if len(out) == 0 {
		return nil
	}
	if len(out) > 1 {
		logrus.Warnf("found %d country candidates for ip %d", len(out), ip)
	}
	return out[0]
}

func (s *CountryStorage) build() error {
	logrus.Info("rebuilding the storage...")

	gStartTSNano := time.Now().UnixNano()

	logrus.Debug("extracting the data...")

	startTSNano := time.Now().UnixNano()

	defer s.dataSource.Cleanup()
	err := s.dataSource.Load()
	if err != nil {
		return err
	}
	logrus.Debugf("done, took %v sec", float64(time.Now().UnixNano()-startTSNano)/float64(time.Second))
	logrus.Print("building the tree...")

	startTSNano = time.Now().UnixNano()

	countries := s.dataSource.GetCountries()
	countriesMap := make(map[uint64]Country)
	for _, c := range countries {
		if _, ok := countriesMap[c.ID]; ok {
			logrus.Warnf("duplicate country id: %d, skipping", c.ID)
		}

		countriesMap[c.ID] = c
	}

	treeMap := make(map[uint32]map[uint32]*Country)
	for _, b := range s.dataSource.GetIPv4Blocks() {
		if _, ok := treeMap[b.StartIP]; !ok {
			treeMap[b.StartIP] = make(map[uint32]*Country)
		}
		if country, ok := countriesMap[b.CountryID]; ok {
			treeMap[b.StartIP][b.EndIP] = &country
		} else {
			logrus.Warnf("country with country id %d does not exist", b.CountryID)
		}
	}

	t := btree.New(2)
	for startIP, ends := range treeMap {
		et := btree.New(2)
		for endIP, country := range ends {
			et.ReplaceOrInsert(&btree.Item{
				Key:     endIP,
				Payload: country,
			})
		}
		rs := &btree.Item{
			Key:     startIP,
			SubTree: et,
		}
		t.ReplaceOrInsert(rs)
	}
	s.tree = t

	logrus.Debugf("done, took %v sec", float64(time.Now().UnixNano()-startTSNano)/float64(time.Second))
	logrus.Infof("extracted & rebuilt, took %v sec", float64(time.Now().UnixNano()-gStartTSNano)/float64(time.Second))

	return nil
}
