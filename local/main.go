package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"mysocks"
	"net"
	"net/http"
	"strconv"
)

type LocalServer struct {
	LocalListenAddr  *net.TCPAddr
	RemoteServerAddr *net.TCPAddr
}

func main() {
	// init local config
	localListenAddr, err := net.ResolveTCPAddr("tcp", mysocks.LocalListenAddr)
	if err != nil {
		log.Fatal(err)
	}
	remoteServerAddr, err := net.ResolveTCPAddr("tcp", mysocks.RemoteServerAddr)
	if err != nil {
		log.Fatal(err)
	}
	localServer := &LocalServer{
		LocalListenAddr:  localListenAddr,
		RemoteServerAddr: remoteServerAddr,
	}

	// listen to local port
	listenLocal, err := net.ListenTCP("tcp", localListenAddr)
	if err != nil {
		log.Fatal(err)
	}

	for {
		localConn, err := listenLocal.AcceptTCP()
		if err != nil {
			fmt.Println(err)
			continue
		}

		// async handle this tcp link
		go localServer.handleLocalConn(localConn)
	}
}

func (l *LocalServer) handleLocalConn(localConn *net.TCPConn) {
	defer localConn.Close()
	// link to remote server
	remoteConn, err := net.DialTCP("tcp", nil, l.RemoteServerAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer remoteConn.Close()

	// first read, send target addr
	frbuf := make([]byte, mysocks.BufSize)
	fwbuf := make([]byte, 256)
	rcount, err := localConn.Read(frbuf)
	if err != nil {
		fmt.Println(err)
		return
	}
	bufreader := bytes.NewReader(frbuf)
	bufioreader := bufio.NewReader(bufreader)
	req, err := http.ReadRequest(bufioreader)
	if err != nil {
		fmt.Println("Parse to req failed\n", string(frbuf), err)
		return
	}
	url := []byte(req.URL.Hostname())
	strport := req.URL.Port()
	port, err := strconv.Atoi(strport)
	port16 := uint16(port)
	portBuffer := bytes.NewBuffer([]byte{})
	binary.Write(portBuffer, binary.BigEndian, port16)
	bport := portBuffer.Bytes()
	copy(fwbuf[:len(url)], url)
	copy(fwbuf[len(url):len(url)+2], bport)
	remoteConn.Write(fwbuf[:len(url)+2])

	// first get
	getCount, err := remoteConn.Read(fwbuf)
	if err != nil {
		fmt.Println("first get err: ", err)
		return
	}
	if getCount == 0 || fwbuf[0] != 0x01 {
		fmt.Print("TCP with target err\n")
		return
	}
	// real first send
	if req.Method == "CONNECT" {
		fmt.Fprint(localConn, "HTTP/1.1 200 Connection established\r\n\r\n")
	} else {
		remoteConn.Write(frbuf[:rcount])
	}

	// process receive
	go func() {
		err = mysocks.Copy(remoteConn, localConn)
		if err != nil {
			localConn.Close()
			remoteConn.Close()
		}
	}()
	// local to remote
	mysocks.Copy(localConn, remoteConn)
}
