#!/bin/bash

# This runs inside the contianer and copies the generated binaries 
# to the host mounted directory.
# Note: the container must be run with a volume mounted 

cp $PROJ/dhcp-go/dhcprelay $PROJ/bin/
cp $PROJ/icpi/bin/libicpi.so $PROJ/bin/

