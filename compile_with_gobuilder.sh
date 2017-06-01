#!/bin/bash

# The PROJ is defined in Dockerfile.build and is set to root project directory
# This file runs inside the container

cd $PROJ/icpi
make clean; make
cd $PROJ

cd $PROJ/dhcp-go

CGO_CPPFLAGS="-I $PROJ/icpi/include" CGO_LDFLAGS="-L $PROJ/icpi/bin" GO_ENABLED=0 GOOS=linux go build -tags netgo -a -installsuffix cgo -o dhcprelay main.go dhcp_relay_icpi.go dhcp_relay_utils.go dhcp_relay_kafka.go

cp $PROJ/dhcp-go/dhcprelay $PROJ/bin/
