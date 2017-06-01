#!/bin/bash
echo $GOPATH
cd icpi
make
make install

cd ../dhcp-go
go get golang.org/x/net/ipv4
go get github.com/krolaw/dhcp4
go get github.com/google/gopacket
go get github.com/google/gopacket/layers
go get gopkg.in/Shopify/sarama.v1

go build -o dhcprelay main.go dhcp_relay_icpi.go dhcp_relay_utils.go dhcp_relay_kafka.go

