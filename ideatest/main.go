package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"mysocks"
	"net"
	"net/http"
	"strconv"
)

func main() {
	listenAddr, err := net.ResolveTCPAddr("tcp", mysocks.LocalListenAddr)
	if err != nil {
		log.Fatal(err)
	}
	listen, err := net.ListenTCP("tcp", listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer listen.Close()
	for {
		conn, err := listen.AcceptTCP()
		if err != nil {
			continue
		}

		go handleConn(conn)
	}
}

func handleConn(conn *net.TCPConn) {
	defer conn.Close()

	buf := make([]byte, mysocks.BufSize)
	rcount, err := conn.Read(buf)
	if err != nil {
		fmt.Println(err)
		return
	}
	bufreader := bytes.NewReader(buf)
	bufioreader := bufio.NewReader(bufreader)
	req, err := http.ReadRequest(bufioreader)
	method := req.Method
	url := req.URL.Hostname()
	port, err := strconv.Atoi(req.URL.Port())
	ipAddr, err := net.ResolveIPAddr("ip", url)
	dIP := ipAddr.IP
	dstAddr := &net.TCPAddr{
		IP:   dIP,
		Port: port,
	}

	targetConn, err := net.DialTCP("tcp", nil, dstAddr)
	if err != nil {
		fmt.Println("Dial target TCP err: ", err)
		return
	}
	defer targetConn.Close()
	if method == "CONNECT" {
		fmt.Fprint(conn, "HTTP/1.1 200 Connection established\r\n\r\n")
	} else {
		targetConn.Write(buf[:rcount])
	}

	go func() {
		err := mysocks.EncodeAndDecodeCopy(conn, targetConn)
		if err != nil {
			conn.Close()
			targetConn.Close()
			return
		}
	}()
	mysocks.EncodeAndDecodeCopy(targetConn, conn)
}
