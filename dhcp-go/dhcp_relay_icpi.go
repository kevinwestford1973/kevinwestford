package main

import (
    "fmt"
    "net"
    "strconv"
    "encoding/hex"
    "cmts-l3-dhcprelay/icpi/gosrc"
    "golang.org/x/net/ipv4"
)

type serveIfConn struct {
    cnrIndex   int
    entryIndex uint32
    pconn      *ipv4.PacketConn
    cm         *ipv4.ControlMessage
    nconn      *net.PacketConn
}

/*
 * Function used to dump punt DHCP packets
 */
func puntHandlerDhcp(cause int, pkt []byte, length int) {
    punt_pkts++
    fmt.Println("Punt Cause:", cause, "length:", length, "punt_pkts", punt_pkts)
    fmt.Println(hex.Dump(pkt))
    PuntDhcpReinject(pkt, length, relayType)
}

func InitIcpi(local_ip, peer_ip string) (uint32, error) {    
    err := icpi.IcpiInitService(local_ip)
    if err != nil {
        fmt.Println(err)
	return 0, err
    }

    for pc := icpi.ICPI_PUNT_CAUSE_UNKNOWN; pc < icpi.ICPI_PUNT_CAUSE_LAST; pc++ {
        err := icpi.IcpiRegisterPuntCause(pc, puntHandlerDhcp)
        if err != nil {
            fmt.Println(err)
	    return 0, err
	}
    }

    var entry_idx uint32
    err = icpi.IcpiRegisterInjectService(peer_ip, "DHCP_INJECT", &entry_idx)
    if err != nil {
        fmt.Println(err)
	return 0 ,err
    }
    
    return entry_idx, nil;
}

func InitConnection (interfaceName string, entryIndex uint32, port int) error {
     fmt.Printfln("interface name", interfaceName);
    iface, err := net.InterfaceByName(interfaceName)
    if err != nil {
        return err
    }
    fmt.Println("listen on ", interfaceName, iface.Index, port)
    
    addrs, err := iface.Addrs()
    if err != nil {
	return err
    }
    
    var ipaddr string
    for _, addr := range addrs {
        switch v := addr.(type) {
        case *net.IPNet:
            if v.IP.DefaultMask() != nil {
                fmt.Println(v.IP)
                ipaddr = v.IP.String()
            }
        }
    }

    p := strconv.Itoa(port)
    fmt.Printf("%s", ipaddr)
    l, err := net.ListenPacket("udp4", ipaddr+":"+p)
    if err != nil {
        return err
    }

    return initConn(entryIndex, iface.Index, l)
}

func initConn(entryIndex uint32, cnrIndex int, conn net.PacketConn) error {
    p := ipv4.NewPacketConn(conn)
    if err := p.SetControlMessage(ipv4.FlagInterface, true); err != nil {
        return err
    }
    Connection.entryIndex = entryIndex
    Connection.cnrIndex = cnrIndex
    Connection.pconn = p
    Connection.nconn = &conn

    return nil
}


