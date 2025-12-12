package valueobjects

import (
	"encoding/base64"
	"fmt"
)

const (
	MethodAES256GCM            = "aes-256-gcm"
	MethodAES128GCM            = "aes-128-gcm"
	MethodAES128CFB            = "aes-128-cfb"
	MethodAES192CFB            = "aes-192-cfb"
	MethodAES256CFB            = "aes-256-cfb"
	MethodAES128CTR            = "aes-128-ctr"
	MethodAES192CTR            = "aes-192-ctr"
	MethodAES256CTR            = "aes-256-ctr"
	MethodChacha20IETF         = "chacha20-ietf"
	MethodChacha20IETFPoly1305 = "chacha20-ietf-poly1305"
	MethodXChacha20IETFPoly1305 = "xchacha20-ietf-poly1305"
	MethodRC4MD5               = "rc4-md5"
)

var validMethods = map[string]bool{
	MethodAES256GCM:             true,
	MethodAES128GCM:             true,
	MethodAES128CFB:             true,
	MethodAES192CFB:             true,
	MethodAES256CFB:             true,
	MethodAES128CTR:             true,
	MethodAES192CTR:             true,
	MethodAES256CTR:             true,
	MethodChacha20IETF:          true,
	MethodChacha20IETFPoly1305:  true,
	MethodXChacha20IETFPoly1305: true,
	MethodRC4MD5:                true,
}

type EncryptionConfig struct {
	method string
}

func NewEncryptionConfig(method string) (EncryptionConfig, error) {
	if !isValidMethod(method) {
		return EncryptionConfig{}, fmt.Errorf("unsupported encryption method: %s", method)
	}

	return EncryptionConfig{
		method: method,
	}, nil
}

func (ec EncryptionConfig) Method() string {
	return ec.method
}

// ToShadowsocksURI generates the Shadowsocks URI with the given password
// The password parameter should be the subscription UUID
func (ec EncryptionConfig) ToShadowsocksURI(password string) string {
	auth := fmt.Sprintf("%s:%s", ec.method, password)
	return base64.URLEncoding.EncodeToString([]byte(auth))
}

func (ec EncryptionConfig) Equals(other EncryptionConfig) bool {
	return ec.method == other.method
}

func isValidMethod(method string) bool {
	return validMethods[method]
}
