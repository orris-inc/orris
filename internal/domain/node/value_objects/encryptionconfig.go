package value_objects

import (
	"encoding/base64"
	"fmt"
)

const (
	MethodAES256GCM            = "aes-256-gcm"
	MethodAES128GCM            = "aes-128-gcm"
	MethodChacha20IETFPoly1305 = "chacha20-ietf-poly1305"
)

var validMethods = map[string]bool{
	MethodAES256GCM:            true,
	MethodAES128GCM:            true,
	MethodChacha20IETFPoly1305: true,
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
