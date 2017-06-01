#!/bin/bash

# Generate the container that has go packages for build purpose
docker build -t ccmts:gobuilder -f Dockerfile.build .

sleep 1

# Run the container with voulme mounted so that the binaries can be copied to the host mounted folder
rm -rf bin; docker run -v $PWD/bin:/root/src/cmts-l3-dhcprelay/bin:Z ccmts:gobuilder
