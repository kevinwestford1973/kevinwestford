package main

import (
    "fmt"
    dhcp "github.com/krolaw/dhcp4"
    "net"
    gopacket "github.com/google/gopacket"
    "github.com/google/gopacket/layers"
    "cmts-l3-dhcprelay/icpi/gosrc"
    "encoding/hex"
)

/*
 * Function used to glean and insert options for punt DHCP packets
 * p dhcp.Packet is punt dhcp packet
 */
func ccmtsServeDhcp (p *dhcp.Packet, msgType dhcp.MessageType, relayType int) (dropPkt bool, retValue bool) {
    if p == nil {
        return true,false
    }

    /* CCMTS glean here */
    

    /*
     * Layer 3 relay special handling
     */ 
    if relayType == DhcpL3Relay {
        /* Boot Request set GIAddr */
        if p.OpCode() == 1 {
            p.SetGIAddr(dhcpGIAddr)
        }
    }

    /* Options insert part starts here */
    /* Test code insert  option43.10 Vendor name */
    p.AddOption(dhcp.OptionVendorSpecificInformation, []byte {0x0a, 0x05, 0x43, 0x43, 0x4d, 0x54, 0x53})
    /* Options insert part end here */

    
    /* Send Kafka message to update CM state */
    PubNetstateCM(p.CHAddr(), p.YIAddr(), msgType);

    return false, true
}

/*
 * Function used for DHCP packet handling
 * relayType 1, layer 2 relay, only insert options
 * relayType 2, layer 3 relay, insert options, set GIAddr and send to dhcp server
 */
func ServeDHCP (p dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options, relayType int) (d dhcp.Packet) {
    var pNew dhcp.Packet
    switch msgType {
    case dhcp.Discover:
        fmt.Println("Discover ", p.YIAddr(), "from", p.CHAddr())
        pNew = dhcp.NewPacket(dhcp.BootRequest)
        pNew.SetCHAddr(p.CHAddr())
        pNew.SetXId(p.XId())
        pNew.SetBroadcast(false)
        for k, v := range p.ParseOptions() {
            pNew.AddOption(k, v)
        }
        break

    case dhcp.Offer:
        /* No related XId found */
        fmt.Println("Offer from", p.YIAddr(), "to", p.CHAddr())
        /* Glean options here */
        pNew = dhcp.NewPacket(dhcp.BootReply)
        pNew.SetXId(p.XId())
        pNew.SetFile(p.File()) 
        pNew.SetFlags(p.Flags()) 
	pNew.SetYIAddr(p.YIAddr()) 
	pNew.SetGIAddr(p.GIAddr()) 
	pNew.SetSIAddr(p.SIAddr()) 
	pNew.SetCHAddr(p.CHAddr()) 
	pNew.SetSecs(p.Secs()) 
	for k, v := range p.ParseOptions() { 
            pNew.AddOption(k, v) 
	} 
        break

    case dhcp.Request:
	fmt.Println("Request ", p.YIAddr(), "from", p.CHAddr()) 
	pNew = dhcp.NewPacket(dhcp.BootRequest) 
	pNew.SetCHAddr(p.CHAddr()) 
	pNew.SetFile(p.File()) 
	pNew.SetCIAddr(p.CIAddr()) 
	pNew.SetSIAddr(p.SIAddr())
	pNew.SetXId(p.XId()) 
	pNew.SetBroadcast(false) 
	for k, v := range p.ParseOptions() { 
	    pNew.AddOption(k, v) 
	} 
        break
 
    case dhcp.ACK: 
 	fmt.Println("ACK from", p.YIAddr(), "to", p.CHAddr()) 
 	pNew = dhcp.NewPacket(dhcp.BootReply) 
 	pNew.SetXId(p.XId()) 
 	pNew.SetFile(p.File()) 
 	pNew.SetFlags(p.Flags()) 
	pNew.SetSIAddr(p.SIAddr()) 
	pNew.SetYIAddr(p.YIAddr()) 
	pNew.SetGIAddr(p.GIAddr()) 
	pNew.SetCHAddr(p.CHAddr()) 
	pNew.SetSecs(p.Secs()) 
	for k, v := range p.ParseOptions() { 
	    pNew.AddOption(k, v) 
	} 
        break
 
    case dhcp.NAK: 
	fmt.Println("NAK from", p.SIAddr(), p.YIAddr(), "to", p.CHAddr()) 
	pNew = dhcp.NewPacket(dhcp.BootReply) 
	pNew.SetXId(p.XId()) 
	pNew.SetFile(p.File()) 
	pNew.SetFlags(p.Flags()) 
        pNew.SetSIAddr(p.SIAddr()) 
	pNew.SetYIAddr(p.YIAddr()) 
	pNew.SetGIAddr(p.GIAddr()) 
	pNew.SetCHAddr(p.CHAddr()) 
        pNew.SetSecs(p.Secs()) 
	for k, v := range p.ParseOptions() { 
	    pNew.AddOption(k, v) 
	} 
        break

    default:
        fmt.Println("Unkown Type ", msgType, "from ", p.CHAddr(), "may support later")
        return nil 
    }

    dropPkt, _ := ccmtsServeDhcp(&pNew, msgType, relayType)
    if dropPkt == true {
        return nil
    }
    
    return pNew
}


/*
 * Function used to insert cable options into punt dhcp packets
 * Discover, Request
 * Input: buffer must be packet with ethernet header
 */
func PuntDhcpReinject(buffer []byte, length int, relayType int) {
    var eth layers.Ethernet
    var ip4 layers.IPv4
    var udp layers.UDP
    var reqType dhcp.MessageType

    if length == 0 {
        return
    }
    
    /* Input must be DHCP */
    req := dhcp.Packet(buffer[42:length])
    if req.HLen() > 16 {
        /* Invalid size of hw len */
        return
    }

    options := req.ParseOptions()
    /* Confirm a DHCP first, then check DHCP type */
    if t := options[dhcp.OptionDHCPMessageType]; len(t) != 1 {
        return
    } else {
        reqType = dhcp.MessageType(t[0])
        if reqType < dhcp.Discover || reqType > dhcp.Inform {
            return
        }
    }

    if res := ServeDHCP(req, reqType, options, relayType); res != nil {
        if res.OpCode() == 1 {
            fmt.Println("Parse punt Boot Request packet eth, ipv4, udp headers")
        } else {
            fmt.Println("Parse punt Boot Reply packet eth, ipv4, udp headers") 
        }
            
        parser := gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet, &eth, &ip4, &udp)
        decodedLayers := []gopacket.LayerType{}
        parser.DecodeLayers(buffer, &decodedLayers)

        udp.SetNetworkLayerForChecksum(&ip4)

        /* Encode a new pkt to inject to VPP-DP */
        injbuf := gopacket.NewSerializeBuffer()
        opts := gopacket.SerializeOptions{
            FixLengths: true,
            ComputeChecksums: true,
        }

        if relayType == DhcpL3Relay {
            if res.OpCode() == 1 {
                /* Boot Request handling */
                for _, ip := range dhcpServers {
                    ip4.TTL = 255
                    udp.DstPort = 67
                    udp.SrcPort = 67
                    ip4.DstIP = ip
                    ip4.SrcIP = dhcpGIAddr;
                    eth.DstMAC = nextHopMAC
                    eth.SrcMAC = dhcpRelayIf.HardwareAddr
                    payload := gopacket.Payload(res)
                    gopacket.SerializeLayers(injbuf, opts, &eth, &ip4, &udp, payload)
                    buf := injbuf.Bytes()
                    icpi.IcpiInjectPacket(Connection.entryIndex, DhcpUSReinject, buf, len(buf))
                    fmt.Println(hex.Dump(buf))
                }
                return
            } else {
                /* Boot reply handling */
                ip4.DstIP = net.ParseIP("255.255.255.255")
                ip4.SrcIP = dhcpGIAddr
                udp.DstPort = 68
                udp.SrcPort = 67
                if res.Broadcast() {
                    eth.DstMAC, _ = net.ParseMAC("ff:ff:ff:ff:ff:ff")
                } else {
                    eth.DstMAC = res.CHAddr()
                }
                eth.SrcMAC = dhcpRelayIf.HardwareAddr
                
            }
        } else {
            /* DhcpL2Relay */
            //if res.OpCode() == 2 { // WR for inject flooding issue
            //    eth.SrcMAC, _= net.ParseMAC("00:0c:29:fa:11:26")
            //}
        }

        payload := gopacket.Payload(res)
        gopacket.SerializeLayers(injbuf, opts, &eth, &ip4, &udp, payload)
        /* DHCP reinject */
        buf := injbuf.Bytes()
        if res.OpCode() == 1 {
            /* Boot Request handling */
            icpi.IcpiInjectPacket(Connection.entryIndex, DhcpUSReinject, buf, len(buf))
        } else {
            /* Boot Reply handling */
            icpi.IcpiInjectPacket(Connection.entryIndex, DhcpDSReinject, buf, len(buf))
        }
        fmt.Println(hex.Dump(buf))
    }
    return
}
