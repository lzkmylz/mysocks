package main

import (
	"fmt"
	"log"
	"mysocks"
	"net"
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
	localConn.SetLinger(0)
	// link to remote server
	remoteConn, err := net.DialTCP("tcp", nil, l.RemoteServerAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer remoteConn.Close()
	remoteConn.SetLinger(0)

	// process receive
	go func() {
		err = mysocks.DecodeCopy(remoteConn, localConn)
		if err != nil {
			localConn.Close()
			remoteConn.Close()
		}
	}()
	// local to remote
	mysocks.EncodeCopy(localConn, remoteConn)
}
