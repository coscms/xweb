package xweb

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"log"
)

type Cryptor interface {
	Encode(rawData, authKey string) string
	Decode(cryptedData, authKey string) string
}

type AesCrypto struct {
	key map[string][]byte
}

var DefaultCryptor Cryptor = &AesCrypto{key: make(map[string][]byte, 0)}

const (
	aesKeyLen = 128
	keyLen    = aesKeyLen / 8
)

func (c *AesCrypto) aesKey(key []byte) []byte {
	if c.key == nil {
		c.key = make(map[string][]byte, 0)
	}
	ckey := string(key)
	k, ok := c.key[ckey]
	if !ok {
		if len(key) == keyLen {
			return key
		}

		k = make([]byte, keyLen)
		copy(k, key)
		for i := keyLen; i < len(key); {
			for j := 0; j < keyLen && i < len(key); j, i = j+1, i+1 {
				k[j] ^= key[i]
			}
		}
		c.key[ckey] = k
	}
	return k
}

func (c *AesCrypto) Encode(rawData, authKey string) string {
	in := []byte(rawData)
	key := []byte(authKey)
	key = c.aesKey(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Println(err)
		return ""
	}
	blockSize := block.BlockSize()
	in = PKCS5Padding(in, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	crypted := make([]byte, len(in))
	blockMode.CryptBlocks(crypted, in)
	return Base64Encode(string(crypted))
}

func (c *AesCrypto) Decode(cryptedData, authKey string) string {
	cryptedData = Base64Decode(cryptedData)
	if cryptedData == "" {
		return ""
	}
	in := []byte(cryptedData)
	key := []byte(authKey)
	key = c.aesKey(key)
	block, err := aes.NewCipher(key)
	if err != nil {
		log.Println(err)
		return ""
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(in))
	blockMode.CryptBlocks(origData, in)
	origData = PKCS5UnPadding(origData)
	return string(origData)
}

func ZeroPadding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{0}, padding)
	return append(ciphertext, padtext...)
}

func ZeroUnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	// 去掉最后一个字节 unpadding 次
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}
