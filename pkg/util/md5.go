package util

import (
	"bytes"
	"crypto/md5"
)

func Md5Hash(s string) []byte {
	hash := md5.New()
	hash.Write([]byte(s))
	return hash.Sum(nil)
	//return hex.EncodeToString(hash.Sum(nil))
}

func Md5Equal(a, b []byte) bool {
	return bytes.Compare(a, b) == 0
}
