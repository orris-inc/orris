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
	method   string
	password string
}

func NewEncryptionConfig(method, password string) (EncryptionConfig, error) {
	if !isValidMethod(method) {
		return EncryptionConfig{}, fmt.Errorf("unsupported encryption method: %s", method)
	}

	if len(password) < 8 {
		return EncryptionConfig{}, fmt.Errorf("password must be at least 8 characters long")
	}

	return EncryptionConfig{
		method:   method,
		password: password,
	}, nil
}

func (ec EncryptionConfig) Method() string {
	return ec.method
}

func (ec EncryptionConfig) Password() string {
	return ec.password
}

func (ec EncryptionConfig) ToShadowsocksURI() string {
	auth := fmt.Sprintf("%s:%s", ec.method, ec.password)
	return base64.URLEncoding.EncodeToString([]byte(auth))
}

func (ec EncryptionConfig) Equals(other EncryptionConfig) bool {
	return ec.method == other.method && ec.password == other.password
}

func isValidMethod(method string) bool {
	return validMethods[method]
}
