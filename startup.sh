#!/bin/sh

# Partial name should be enough for matching
VETH_IF=veth_

ICPI_CONF=/etc/icpi.conf
DEFAULT_ICPI_LOCAL_IP=0.0.0.0
DEFAULT_ICPI_LOCAL_PORT=9002
DEFAULT_ICPI_REMOTE_IP=10.10.1.2
DEFAULT_ICPI_REMOTE_PORT=9003

# Wait for veth interface to be created
while true; do
    if ip link | grep -q "$VETH_IF"; then
        echo "Interface $VETH_IF has been created"
        sleep 1
        break
    else
        echo "Interface $VETH_IF has not been created, waiting..."
        sleep 1
    fi
done

# Wait for $ICPI_CONF file to be created
while true; do
    if [ -f $ICPI_CONF ]; then
        echo "$ICPI_CONF has been created"
        sleep 2 # sleep 2 seconds to make sure $ICPI_CONF has been filled
        break
    else
        echo "$ICPI_CONF has not been created, waiting..."
        sleep 1
    fi
done

# Parse $ICPI_CONF file
ICPI_LOCAL_IP=`grep ^ICPI_LOCAL_IP= $ICPI_CONF | cut -d= -f 2`
if [ -z $ICPI_LOCAL_IP ]; then
    ICPI_LOCAL_IP=$DEFAULT_ICPI_LOCAL_IP
    echo "ICPI_LOCAL_IP not configured, use default config: $DEFAULT_ICPI_LOCAL_IP"
else
    echo "ICPI_LOCAL_IP configured: $ICPI_LOCAL_IP"
fi

ICPI_REMOTE_IP=`grep ^ICPI_REMOTE_IP= $ICPI_CONF | cut -d= -f 2`
if [ -z $ICPI_REMOTE_IP ]; then
    ICPI_REMOTE_IP=$DEFAULT_ICPI_REMOTE_IP
    echo "ICPI_REMOTE_IP not configured, use default config: $DEFAULT_ICPI_REMOTE_IP"
else
    echo "ICPI_REMOTE_IP configured: $ICPI_REMOTE_IP"
fi

# Start service
./dhcprelay -vppout $ICPI_REMOTE_IP -vppin $ICPI_LOCAL_IP
