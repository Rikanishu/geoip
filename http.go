package main

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	MaxIPsPerRequest = 100
)

const usage = `
Usage:

curl "http://localhost:12950/country/132.99.75.15"

Also you can pass several ip addresses that you need to check:

curl "http://localhost:12950/country/132.99.75.15,99.12.44.52,3.24.12.85"

`

type Server struct {
	config         *Config
	countryStorage *CountryStorage
}

func NewServer(config *Config, countryStorage *CountryStorage) *Server {
	return &Server{
		config:         config,
		countryStorage: countryStorage,
	}
}

func (s *Server) Run() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/", s.usage)
	r.GET("/country/:ips", s.resolveCountry)

	logrus.Infof("starting the HTTP server on %s", s.config.Listen)
	r.Run(s.config.Listen)
}

func (s *Server) usage(c *gin.Context) {
	c.JSON(http.StatusOK, usage)
}

func (s *Server) resolveCountry(c *gin.Context) {
	ips, err := parseIPS(c.Param("ips"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": fmt.Sprintf("%s", err)})
		return
	}
	out := make(map[string]string, len(ips))
	for _, ip := range ips {
		ipInt, err := ipv4toUint32(ip)
		if err != nil {
			continue
		}
		country := s.countryStorage.FindCountry(ipInt)
		if country == nil {
			continue
		}
		out[ip] = country.Code
	}

	c.JSON(http.StatusOK, out)
	return
}

func parseIPS(ips string) ([]string, error) {
	if ips == "" {
		return nil, errors.New("empty ip string passed")
	}

	out := make([]string, 0)
	parts := strings.Split(ips, ",")
	if len(parts) > MaxIPsPerRequest {
		return nil, errors.New("limit of ips in one request reached")
	}
	for _, ip := range parts {
		ip = strings.TrimSpace(ip)
		ipS := net.ParseIP(ip)
		if ipS.To4() == nil {
			return nil, errors.New("not correct ipv4 passed")
		}
		out = append(out, ip)
	}

	if len(out) == 0 {
		return nil, errors.New("has no ip addresses to check")
	}

	return out, nil
}
