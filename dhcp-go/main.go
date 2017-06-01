package main

import (
    "flag"
    "fmt"
    "net"
    "strings"
    "time"
)

const (
    DhcpL2Relay = 1
    DhcpL3Relay = 2
)

const (
    DhcpUSReinject = 6
    DhcpDSReinject = 7
)

var Connection serveIfConn
var dhcpServers []net.IP
var dhcpGIAddr net.IP
var dhcpRelayIf net.Interface
var relayType int
var nextHopMAC net.HardwareAddr
var punt_pkts, inject_pkts int


func createRelay(vppout, vppin, cnr string) {
    entryIndex, err := InitIcpi(vppin, vppout)
    if err != nil {
        fmt.Println(err)
        return
    }
    Connection.entryIndex = entryIndex
}

/*
 * Layer 2 relay, Parameters: NULL
 * Layer 3 relay, Parameters: servers, ifname, nexthop
 */
func main() {
    /* Specify VPP-DP communication IPs */
    flagVppOut := flag.String("vppout", "10.0.3.1", "required ip address for vpp peer ip")
    flagVppIn := flag.String("vppin", "10.0.3.5", "required ip address for vpp bind local ip ")

    /* DHCP Relay parameters */
    flagRelayType := flag.Int("relay", 1, "1: L2-Relay, 2: L3-Relay")
    flagServers := flag.String("servers", "40.0.0.1", "ip1 ip2 (ip addresses of the dhcp servers")
    flagRelayIfname := flag.String("ifname", "ens193", "Layer3 relay interface name")
    flagNextHopMAC := flag.String("nexthop", "00:1e:14:5a:0b:bf", "Layer3 relay next-hop MAC")

    flag.Parse()
    
    relayType = *flagRelayType
    if relayType == DhcpL3Relay {
        /* Set Relay server addresses */
        servers := strings.Fields(*flagServers)
        for _, s := range servers {
	        dhcpServers = append(dhcpServers, net.ParseIP(s))
            fmt.Println("dhcp server", dhcpServers)
        }

        /* Set Relay interface related info */
        intf, err := net.InterfaceByName(*flagRelayIfname)
        if intf == nil {
            fmt.Println("Can't find relay interface ", *flagRelayIfname, err)
        } else {
            dhcpRelayIf = *intf
            /* Get relay interface info */
            ifAddrList, err := dhcpRelayIf.Addrs()
            if err != nil {
                fmt.Println("Can't find relay interface ", *flagRelayIfname, "address ", err)
            }
            /* Get non loopback local IP
             * Set GIAddr
             */
            fmt.Println("Relay interface MAC ",dhcpRelayIf.HardwareAddr.String())
            for _, addrs := range ifAddrList {
                if ipnet, err := addrs.(*net.IPNet); err && !ipnet.IP.IsLoopback() {
                    if ipnet.IP.To4() != nil {
                        dhcpGIAddr = ipnet.IP
                        fmt.Println("GIAddr ", dhcpGIAddr.String())
                        break
                    }
                }
            }
        }
        nextHopMAC, _ = net.ParseMAC(*flagNextHopMAC)
        fmt.Println("Nexthop MAC", nextHopMAC.String())
    }

    createRelay(*flagVppOut, *flagVppIn, "")

    for {
        time.Sleep(100*time.Millisecond)
    }

}
