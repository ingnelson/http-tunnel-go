package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

type ClientManager struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	payload    string
	auth       string
}

type Client struct {
	socket net.Conn
	proxy  Proxy
}

type Proxy struct {
	socket    net.Conn
	connected bool
}

func (manager *ClientManager) start() {
	for {
		select {
		case connection := <-manager.register:
			manager.clients[connection] = true
			log.Println("Added Connection:", connection.socket.RemoteAddr())
		case connection := <-manager.unregister:
			delete(manager.clients, connection)
			log.Println("Remove Connection:", connection.socket.RemoteAddr())
		}
	}
}

func (manager *ClientManager) parsePayload(request *[]byte) []byte {
	if isHttpConnectRequest(request) {
		reqString := string(*request)
		payload := manager.payload                          // copy payload from manager
		splitRequestRaw := strings.Split(reqString, "\r\n") // split http request
		splitRequest := strings.Split(splitRequestRaw[0], " ")

		connHost := splitRequest[1]  // ip after CONNECT
		connProto := splitRequest[2] // http protocol

		parsed := strings.ReplaceAll(payload, "[host_port]", connHost)

		if manager.auth == "" { // authentication
			parsed = strings.ReplaceAll(parsed, "[protocol]", connProto)
		} else {
			parsed = strings.ReplaceAll(parsed, "[protocol]", connProto+"\r\n"+
				"Authorization: Basic "+manager.auth)
		}

		parsed = strings.ReplaceAll(parsed, "[ua]", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/47.0.2526.111 Safari/537.36")
		parsed = strings.ReplaceAll(parsed, "[crlf]", "\r\n")
		parsed = strings.ReplaceAll(parsed, "[cr]", "\r")
		parsed = strings.ReplaceAll(parsed, "[lf]", "\n")

		return []byte(parsed)
	}
	return *request
}

func isHttpConnectRequest(request *[]byte) bool {
	if strings.Contains(string(*request), "CONNECT") {
		return true
	}
	return false
}

func (manager *ClientManager) handleConnection(client *Client) {

	// client to proxy
	go func() {
		size := 32 * 1024
		var r io.Reader = client.socket

		if l, ok := r.(*io.LimitedReader); ok && int64(size) > l.N {
			if l.N < 1 {
				size = 1
			} else {
				size = int(l.N)
			}
		}
		buf := make([]byte, size)

		for {
			nr, err := r.Read(buf)

			if nr > 0 {

				if client.proxy.connected == false {
					p := manager.parsePayload(&buf) // parse connection
					_, err = client.proxy.socket.Write(p)
					if err != nil {
						// cant write to proxy
						log.Println(err)
						break
					}
					client.proxy.connected = true
				} else {
					nw, err := client.proxy.socket.Write(buf[0:nr])
					if err != nil {
						log.Println(err)
						break
					}

					if nr != nw {
						log.Println(err)
						break
					}
				}
			}

			if err != nil {
				if err != io.EOF {
					log.Println(err)
				}
				manager.unregister <- client
				_ = client.proxy.socket.Close()
				break
			}
		}
	}()

	// proxy to client
	go func() {
		size := 32 * 1024
		var r io.Reader = client.proxy.socket

		if l, ok := r.(*io.LimitedReader); ok && int64(size) > l.N {
			if l.N < 1 {
				size = 1
			} else {
				size = int(l.N)
			}
		}
		buf := make([]byte, size)

		for {
			nr, err := r.Read(buf)
			if nr > 0 {
				nw, err := client.socket.Write(buf[0:nr])
				if err != nil {
					break
				}
				if nr != nw {
					break
				}
			}

			if err != nil {
				_ = client.socket.Close()
				break
			}
		}
	}()
}

func main() {

	fmt.Println("A Cross-Platform HTTP Tunnel by @lfasmpao | Version: 0.0.1 alpha")

	hostPtr := flag.String("proxy", "", "Proxy Server [host:port] (required)")
	payloadPtr := flag.String("payload", "", "Payload (required)")
	serverPtr := flag.Int("port", 8888, "Server Port")
	authPtr := flag.String("auth", "", "Proxy Authentication [username:password] (optional)")
	flag.Parse()

	if *hostPtr == "" || *payloadPtr == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	log.Println("Server running on port:", *serverPtr)
	log.Println("Server running on proxy:", *hostPtr)
	log.Println("Payload:", *payloadPtr)

	conn, err := net.Listen("tcp", ":"+strconv.Itoa(*serverPtr))

	if err != nil {
		log.Println(err)
	}

	manager := ClientManager{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		payload:    *payloadPtr,
		auth:       base64.StdEncoding.EncodeToString([]byte(*authPtr)),
	}

	go manager.start()

	for {

		accept, err := conn.Accept()
		if err != nil {
			log.Println(err)
		}

		proxy, err := net.Dial("tcp", *hostPtr)
		if err != nil {
			log.Println("Cant connect to proxy")
			continue
		}

		client := &Client{
			socket: accept,
			proxy: Proxy{
				socket:    proxy,
				connected: false,
			},
		}

		manager.register <- client
		go manager.handleConnection(client)

	}

}
