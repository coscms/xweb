package str

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"encoding/json"
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"encoding/base64"
	"strings"
)


// md5 hash string
func Md5(str string) string {
	m := md5.New()
	io.WriteString(m, str)
	return fmt.Sprintf("%x", m.Sum(nil))
}


func Token(key string, val []byte, args ...string) string {
	hm := hmac.New(sha1.New, []byte(key))
	hm.Write(val)
	for _,v := range args {
		hm.Write([]byte(v))
	}
	return fmt.Sprintf("%02x", hm.Sum(nil))
}

func Encode(data interface{}) ([]byte, error) {
	//return JsonEncode(data)
	return GobEncode(data)
}

func Decode(data []byte, to interface{}) error {
	//return JsonDecode(data, to)
	return GobDecode(data, to)
}

func GobEncode(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(&data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func GobDecode(data []byte, to interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(to)
}

func JsonEncode(data interface{}) ([]byte, error) {
	val, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return val, nil
}

func JsonDecode(data []byte, to interface{}) error {
	return json.Unmarshal(data, to)
}

func Base64Encode(val string) string {
	var buf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	encoder.Write([]byte(val))
	encoder.Close()
	return strings.TrimRight(buf.String(), "=")
}

func Base64Decode(val string) string {
	buf := bytes.NewBufferString(val)
	encoder := base64.NewDecoder(base64.StdEncoding, buf)
	res, _ := ioutil.ReadAll(encoder)
	return string(res)
}
