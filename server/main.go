package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"mysocks"
	"net"
)

type RemoteServer struct {
	ListenAddr *net.TCPAddr
}

func main() {
	listenAddr, err := net.ResolveTCPAddr("tcp", mysocks.RemoteServerAddr)
	if err != nil {
		log.Fatal(err)
	}

	remoteServer := &RemoteServer{
		ListenAddr: listenAddr,
	}
	listen, err := net.ListenTCP("tcp", listenAddr)

	for {
		conn, err := listen.AcceptTCP()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go remoteServer.handleConn(conn)
	}
}

func (r *RemoteServer) handleConn(conn *net.TCPConn) {
	buf := make([]byte, mysocks.BufSize)
	defer conn.Close()

	// first get from conn
	rcount, err := conn.Read(buf)
	if err != nil {
		fmt.Println("first read err: ", err)
		return
	}
	url := string(buf[:rcount-2])
	ipAddr, err := net.ResolveIPAddr("ip", url)
	if err != nil {
		fmt.Println("resolve ip err: ", err)
		return
	}
	dIP := ipAddr.IP
	dPort := buf[rcount-2 : rcount]
	port := int(binary.BigEndian.Uint16(dPort))
	//fmt.Println("target is: ", url, ", port: ", port)
	dstAddr := &net.TCPAddr{
		IP:   dIP,
		Port: port,
	}

	// TCP with target
	targetConn, err := net.DialTCP("tcp", nil, dstAddr)
	if err != nil {
		fmt.Println("Dial target TCP err: ", err)
		return
	}
	defer targetConn.Close()

	// to target
	go func() {
		err = mysocks.Copy(conn, targetConn)
		if err != nil {
			targetConn.Close()
			conn.Close()
		}
	}()

	// link success, ack with local server and ask data
	conn.Write([]byte{0x01})
	// to local
	mysocks.Copy(targetConn, conn)
}
