package mysocks

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
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

func EncodeAndDecodeCopy(src, dst *net.TCPConn) error {
	rbuf := make([]byte, 1024)
	encodeBuf := make([]byte, 1536)
	decodeBuf := make([]byte, 1024)

	for {
		rcount, rerr := src.Read(rbuf)
		if rerr != nil {
			if rerr != io.EOF {
				return rerr
			} else {
				return nil
			}
		}

		encodeBlocks := int(math.Floor(float64(rcount) / 256))
		var encodeLength int
		if float64(encodeBlocks) == float64(rcount)/256 {
			encodeLength = 384 * encodeBlocks
		} else {
			encodeLength = 384 * (encodeBlocks + 1)
		}

		for i := 0; i < encodeBlocks; i++ {
			encodeData, encodeErr := RsaEncrypt(rbuf[i*256 : (i+1)*256])
			if encodeErr != nil {
				return encodeErr
			}
			copy(encodeBuf[i*384:(i+1)*384], encodeData)
		}
		if float64(encodeBlocks) != float64(rcount)/256 {
			encodeData, encodeErr := RsaEncrypt(rbuf[encodeBlocks*256 : rcount])
			if encodeErr != nil {
				return encodeErr
			}
			copy(encodeBuf[384*encodeBlocks:384*(encodeBlocks+1)], encodeData)
		}

		decodeBlocks := encodeLength / 384
		decodeLength := 0
		for i := 0; i < decodeBlocks; i++ {
			decodeData, decodeErr := RsaDecrypt(encodeBuf[i*384 : (i+1)*384])
			fmt.Println("decode block length: ", len(decodeData))
			if decodeErr != nil {
				return decodeErr
			}
			if len(decodeData) == 256 {
				decodeLength += 256
				copy(decodeBuf[i*256:(i+1)*256], decodeData)
			} else {
				decodeLength += len(decodeData)
				copy(decodeBuf[i*256:i*256+len(decodeData)], decodeData)
			}
		}

		fmt.Println("before encode and decode length: ", rcount)
		fmt.Println("after encode and decode length: ", decodeLength)
		fmt.Println("before encode and decode: ", rbuf[:rcount])
		fmt.Println("after encode and decode: ", decodeBuf[:decodeLength])

		wcount, werr := dst.Write(decodeBuf[:decodeLength])
		if werr != nil {
			return werr
		}
		if wcount != decodeLength {
			return io.ErrShortWrite
		}
	}
}
