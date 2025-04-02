package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	endpoint, err := net.ResolveUDPAddr("udp", "localhost:42069")
	if err != nil {
		log.Panicf("Error resoling udp addres - %s\n", err)
	}
	conn, err := net.DialUDP(endpoint.Network(), nil, endpoint)
	if err != nil {
		log.Panicf("Error dialing to udp endpoint: %d, - %s\n", endpoint.Port, err)
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(">")
		ln, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		}
		_, err = conn.Write([]byte(ln))
		if err != nil {
			fmt.Println(err)
		}
	}
}
