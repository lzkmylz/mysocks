package mysocks

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"math"
	"net"
	"os"
)

func GenRsaKey(bits int) error {
	// 生成私钥文件
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return err
	}
	derStream := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derStream,
	}
	file, err := os.Create("private.pem")
	if err != nil {
		return err
	}
	err = pem.Encode(file, block)
	if err != nil {
		return err
	}
	// 生成公钥文件
	publicKey := &privateKey.PublicKey
	derPkix, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return err
	}
	block = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derPkix,
	}
	file, err = os.Create("public.pem")
	if err != nil {
		return err
	}
	err = pem.Encode(file, block)
	if err != nil {
		return err
	}
	return nil
}

// 从src中源源不断的读取原数据加密后写入到dst，直到src中没有数据可以再读取
func Copy(src, dst *net.TCPConn) error {
	buf := make([]byte, BufSize)
	for {
		readCount, errRead := src.Read(buf)
		if errRead != nil {
			if errRead != io.EOF {
				return errRead
			} else {
				return nil
			}
		}
		if readCount > 0 {
			writeCount, errWrite := dst.Write(buf[:readCount])
			if errWrite != nil {
				return errWrite
			}
			if readCount != writeCount {
				return io.ErrShortWrite
			}
		}
	}
}

func EncodeCopy(src, dst *net.TCPConn) error {
	rbuf := make([]byte, 1024)
	wbuf := make([]byte, 1536)

	for {
		rcount, errRead := src.Read(rbuf)
		if errRead != nil {
			if errRead != io.EOF {
				return errRead
			} else {
				return nil
			}
		}

		blocks := int(math.Ceil(float64(rcount) / 256))
		for i := 0; i < blocks; i++ {
			encodeData, encodeErr := RsaEncrypt(rbuf[256*i : 256*(i+1)])
			if encodeErr != nil {
				return encodeErr
			}
			copy(wbuf[384*i:384*(i+1)], encodeData)
		}
		wcount, werr := dst.Write(wbuf[:blocks*384])
		if werr != nil {
			return werr
		}
		if wcount != blocks*384 {
			return io.ErrShortWrite
		}
	}
}

func DecodeCopy(src, dst *net.TCPConn) error {
	rbuf := make([]byte, 1536)
	wbuf := make([]byte, 1024)

	for {
		rcount, errRead := src.Read(rbuf)
		if errRead != nil {
			if errRead != io.EOF {
				return errRead
			} else {
				return nil
			}
		}

		blocks := int(math.Ceil(float64(rcount) / 384))
		for i := 0; i < blocks; i++ {
			decodeData, decodeErr := RsaDecrypt(rbuf[384*i : 384*(i+1)])
			if decodeErr != nil {
				return decodeErr
			}
			copy(wbuf[i*256:(i+1)*256], decodeData)
		}

		wcount, werr := dst.Write(wbuf[:blocks*256])
		if werr != nil {
			return werr
		}
		if wcount != blocks*256 {
			return io.ErrShortWrite
		}
	}
}
