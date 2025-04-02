package main

import (
	"fmt"
	"log"
	"net"

	"github.com/Barrioslopezfd/httpfromtcp/internal/request"
)

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		log.Panicf("Error listening - Err=%s\n", err)
	}
	fmt.Print("Listening to port 42069\n")
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Panic(err)
		}
		fmt.Print("Connection accepted\n")

		r, err := request.RequestFromReader(conn)
		if err != nil {
			log.Panic(err)
		}

		fmt.Printf("Request line:\n- Method: %s\n- Target: %s\n- Version: %s\nHeaders:\n", r.RequestLine.Method, r.RequestLine.RequestTarget, r.RequestLine.HttpVersion)
		for key, value := range r.Headers {
			fmt.Printf("- %s: %s\n", key, value)
		}
		if len(r.Body) != 0 {
			fmt.Printf("Body:\n%s\n", string(r.Body))
		}

		fmt.Print("Connection closed\n")
	}
}
