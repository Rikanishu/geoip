#!/usr/bin/env bash

docker build -t "geoip" ./ && docker run --name "geoip" --rm -p 12950:12950 -p 12951:12951 "geoip"
