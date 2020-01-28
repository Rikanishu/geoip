package main

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	DBUrl                       = "https://geolite.maxmind.com/download/geoip/database/GeoLite2-Country-CSV.zip"
	CountriesIPV4BlocksFileName = "GeoLite2-Country-Blocks-IPv4.csv"
	CountriesENFileName         = "GeoLite2-Country-Locations-en.csv"

	countriesInitCount = 250
	blocksInitCount    = 5000

	defaultDownloadTimeout = 1 * time.Minute
)

type respCSVData struct {
	countriesEN []byte
	blocksIPv4  []byte
}

type MaxmindRemoteCountryDateSource struct {
	config     *Config
	countries  []Country
	blocksIPv4 []IPv4CountryBlock
}

func NewMaxmindRemoteCountryDateSource(conf *Config) *MaxmindRemoteCountryDateSource {
	return &MaxmindRemoteCountryDateSource{
		config: conf,
	}
}

func (s *MaxmindRemoteCountryDateSource) Load() error {
	logrus.Debug("start geoip countries downloading")

	resp, err := s.download()
	if err != nil {
		return err
	}
	err = s.parseCountries(resp)
	if err != nil {
		return err
	}
	err = s.parseIPV4Blocks(resp)
	if err != nil {
		return err
	}

	logrus.Debugf("got %d countries, %d blocks", len(s.countries), len(s.blocksIPv4))

	return nil
}

func (s *MaxmindRemoteCountryDateSource) GetCountries() []Country {
	return s.countries
}

func (s *MaxmindRemoteCountryDateSource) GetIPv4Blocks() []IPv4CountryBlock {
	return s.blocksIPv4
}

func (s *MaxmindRemoteCountryDateSource) SupportUpdates() bool {
	return true
}

func (s *MaxmindRemoteCountryDateSource) GetNextUpdateTime() time.Time {
	return time.Now().Add(7 * 24 * time.Hour)
}

func (s *MaxmindRemoteCountryDateSource) Cleanup() error {
	s.countries = nil
	s.blocksIPv4 = nil

	return nil
}

func (s *MaxmindRemoteCountryDateSource) download() (*respCSVData, error) {

	client := &http.Client{
		Timeout: defaultDownloadTimeout,
	}
	resp, err := client.Get(DBUrl)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get coutnries data")
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read response bytes")
	}

	logrus.Debugf("downloaded zip file, %d bytes", len(content))

	archive, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, errors.Wrap(err, "unbale to open zip archive")
	}

	out := new(respCSVData)
	for _, f := range archive.File {
		isCountriesEN := strings.HasSuffix(f.Name, CountriesENFileName)
		isBlocksIPV4 := strings.HasSuffix(f.Name, CountriesIPV4BlocksFileName)
		if isCountriesEN || isBlocksIPV4 {
			rc, err := f.Open()
			if err != nil {
				return nil, errors.Wrap(err, "can't open file in archive")
			}
			bs, err := ioutil.ReadAll(rc)
			if err != nil {
				//todo: add defer and move it to another function
				rc.Close()
				return nil, errors.Wrap(err, "can't read file in archive")
			}
			if isCountriesEN {
				out.countriesEN = bs
			} else if isBlocksIPV4 {
				out.blocksIPv4 = bs
			}
			//todo: add defer and move it to another function
			rc.Close()
		}
	}

	return out, nil
}

func (s *MaxmindRemoteCountryDateSource) parseCountries(d *respCSVData) error {
	if len(d.countriesEN) == 0 {
		return errors.New("countries data file is not loaded")
	}
	b := bytes.NewReader(d.countriesEN)
	r := csv.NewReader(b)
	countries := make([]Country, 0, countriesInitCount)
	k := 0
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "CSV reading error")
		}
		if k != 0 {
			id, err := strconv.ParseUint(record[0], 10, 64)
			if err != nil {
				return errors.Wrap(err, "unable to parse geoname id")
			}
			countries = append(countries, Country{
				ID:    id,
				Code:  record[4],
				Title: record[5],
			})
		}
		k++
	}
	s.countries = countries

	return nil
}

func (s *MaxmindRemoteCountryDateSource) parseIPV4Blocks(d *respCSVData) error {
	if len(d.blocksIPv4) == 0 {
		return errors.New("countries data file is not loaded")
	}
	b := bytes.NewReader(d.blocksIPv4)
	r := csv.NewReader(b)
	blocks := make([]IPv4CountryBlock, 0, blocksInitCount)
	k := 0
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "CSV reading error")
		}
		if k != 0 && record[1] != "" {
			id, err := strconv.ParseUint(record[1], 10, 64)
			if err != nil {
				return errors.Wrap(err, "unable to parse block geoname id")
			}
			start, end, err := cidrToIPv4Range(record[0])
			if err != nil {
				return errors.Wrap(err, "unable to parse CIDR")
			}
			blocks = append(blocks, IPv4CountryBlock{
				CountryID: id,
				StartIP:   start,
				EndIP:     end,
			})
		}
		k++
	}
	s.blocksIPv4 = blocks

	return nil
}
