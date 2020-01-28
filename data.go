package main

import "time"

type Country struct {
	ID    uint64
	Code  string
	Title string
}

type IPv4CountryBlock struct {
	StartIP   uint32
	EndIP     uint32
	CountryID uint64
}

type CountryDataSource interface {
	Load() error
	GetCountries() []Country
	GetIPv4Blocks() []IPv4CountryBlock
	SupportUpdates() bool
	GetNextUpdateTime() time.Time
	Cleanup() error
}

func BuildCountryDataSource(conf *Config) CountryDataSource {
	return NewMaxmindRemoteCountryDateSource(conf)
}
