package main

import (
    "log"
    "fmt"
    sarama "gopkg.in/Shopify/sarama.v1"
    dhcp "github.com/krolaw/dhcp4"
    "net"
)

var broker = []string {
    "kafka-0.kafka.default.svc.cluster.local:9092",
    "kafka-1.kafka.default.svc.cluster.local:9092",
    "kafka-2.kafka.default.svc.cluster.local:9092",
}

const netstateCMTopic string = "DhcpPubCMStatus"
//const partitionKey string = "netstate"
const partitionKey string = "test"
const client = "NetstateProducer"

func netstateCM (msgType dhcp.MessageType) string {
    switch msgType {
    case dhcp.Discover:
        return "init(d)"
    case dhcp.Offer:
        return "init(io)"
    case dhcp.Request:
        return "init(dr)"
    case dhcp.ACK:
        return "init(i)"
    }
    return ""
}


func PubNetstateCM (mac net.HardwareAddr, ip net.IP, msgType dhcp.MessageType) {
    fmt.Println("Pub msg")
    if netstate := netstateCM(msgType); netstate == "" {
        log.Println("Invalid msgType")
        return
    }
    config := sarama.NewConfig()
    config.ClientID = client
    config.Producer.Return.Successes = true

    producer, err := sarama.NewSyncProducer(broker, config)
    if err != nil {
        log.Println(err)
	return
    }

    /*
     * Client Name: Cable Modem: Net State: Host: Net State: IP Address
     */
    netstateMsg:= "{\"Client Name\": \"Net Service\", " +
                  "\"Cable Modem\": \"" + net.HardwareAddr(mac).String() + "\", " +
                  "\"Net State\": \"" + netstateCM(msgType) + "\", " +
                  "\"IP Address\": \"" + net.IP(ip).String() + "\"}"

    fmt.Println("Kafka MSg", netstateMsg)
    msg := &sarama.ProducerMessage {
        Topic : netstateCMTopic,
        Key   : sarama.StringEncoder(partitionKey),
        Value : sarama.StringEncoder(netstateMsg),
    }

    partition, offset, err := producer.SendMessage(msg)
    if err != nil {
        log.Printf("FAILED to send message: %s\n", err)
    } else {
	log.Printf("CM netstate message sent to partition %d at offset %d\n", partition, offset)
    }

    if err := producer.Close(); err != nil {
        log.Println(err)
    } else {	        
        fmt.Println("Connection closed.")
    }
    fmt.Println("Pub msg OK")
}
