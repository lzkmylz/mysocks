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
	rbuf := make([]byte, 1536)
	decodeBuf := make([]byte, 1024)
	defer conn.Close()
	conn.SetLinger(0)

	// first get from conn
	rcount, err := conn.Read(rbuf)
	if err != nil {
		fmt.Println("first read err: ", err)
		return
	}

	// decode first read
	decodeBlocks := rcount / 384
	decodeLength := 0
	for i := 0; i < decodeBlocks; i++ {
		decodeData, decodeErr := mysocks.RsaDecrypt(rbuf[i*384 : (i+1)*384])
		if decodeErr != nil {
			fmt.Println("first read, decode block error, ", decodeErr)
			return
		}
		if len(decodeData) == 256 {
			decodeLength += 256
			copy(decodeBuf[i*256:(i+1)*256], decodeData)
		} else {
			decodeLength += len(decodeData)
			copy(decodeBuf[i*256:i*256+len(decodeData)], decodeData)
		}
	}

	decodeBufReader := bytes.NewReader(decodeBuf[:decodeLength])
	decodeBufioReader := bufio.NewReader(decodeBufReader)
	req, err := http.ReadRequest(decodeBufioReader)
	if err != nil {
		fmt.Println("parse requrest error", err)
		return
	}
	url := req.URL.Hostname()
	port, err := strconv.Atoi(req.URL.Port())
	if err != nil {
		fmt.Println("parse request port error, ", err)
		fmt.Println("original request:\n", string(decodeBuf[:decodeLength]))
		return
	}
	ip, err := net.ResolveIPAddr("ip", url)
	if err != nil {
		fmt.Println("parse url to ip err", err)
		fmt.Println("use url: ", url)
		fmt.Println("original request:\n", string(decodeBuf[:decodeLength]))
		return
	}
	dstAddr := &net.TCPAddr{
		IP:   ip.IP,
		Port: port,
	}

	targetConn, err := net.DialTCP("tcp", nil, dstAddr)
	if err != nil {
		fmt.Println("Dial to target error", err)
		fmt.Println("original request:\n", string(decodeBuf[:decodeLength]))
		return
	}
	defer targetConn.Close()
	targetConn.SetLinger(0)

	if req.Proto != "HTTP/1.1" {
		fmt.Println("Different Http Proto: ", req.Proto)
	}
	if req.Method == "CONNECT" {
		msg2local := []byte("HTTP/1.1 200 Connection established\r\n\r\n")
		encodeMsg, err := mysocks.RsaEncrypt(msg2local)
		if err != nil {
			fmt.Println("encode connect msg err, ", err)
		}
		conn.Write(encodeMsg)
	} else {
		targetConn.Write(decodeBuf[:decodeLength])
	}

	// to target
	go func() {
		err = mysocks.DecodeCopy(conn, targetConn)
		if err != nil {
			targetConn.Close()
			conn.Close()
		}
	}()
	// to local
	mysocks.EncodeCopy(targetConn, conn)
}
