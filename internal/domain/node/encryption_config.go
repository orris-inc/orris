package node

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const (
	MethodAES256GCM            = "aes-256-gcm"
	MethodAES128GCM            = "aes-128-gcm"
	MethodChacha20IETFPoly1305 = "chacha20-ietf-poly1305"
)

var supportedMethods = map[string]bool{
	MethodAES256GCM:            true,
	MethodAES128GCM:            true,
	MethodChacha20IETFPoly1305: true,
}

type EncryptionConfig struct {
	method   string
	password string
}

func NewEncryptionConfig(method, password string) (*EncryptionConfig, error) {
	normalizedMethod := strings.ToLower(strings.TrimSpace(method))

	if !isValidMethod(normalizedMethod) {
		return nil, fmt.Errorf("unsupported encryption method: %s", method)
	}

	if len(password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters long")
	}

	if len(password) > 128 {
		return nil, fmt.Errorf("password cannot exceed 128 characters")
	}

	return &EncryptionConfig{
		method:   normalizedMethod,
		password: password,
	}, nil
}

func (ec *EncryptionConfig) Method() string {
	return ec.method
}

func (ec *EncryptionConfig) Password() string {
	return ec.password
}

func (ec *EncryptionConfig) ToShadowsocksAuth() string {
	auth := fmt.Sprintf("%s:%s", ec.method, ec.password)
	return base64.URLEncoding.EncodeToString([]byte(auth))
}

func (ec *EncryptionConfig) Equals(other *EncryptionConfig) bool {
	if ec == nil || other == nil {
		return ec == other
	}
	return ec.method == other.method && ec.password == other.password
}

func (ec *EncryptionConfig) IsAES() bool {
	return ec.method == MethodAES256GCM || ec.method == MethodAES128GCM
}

func (ec *EncryptionConfig) IsChacha20() bool {
	return ec.method == MethodChacha20IETFPoly1305
}

func isValidMethod(method string) bool {
	return supportedMethods[method]
}

func GetSupportedMethods() []string {
	methods := make([]string, 0, len(supportedMethods))
	for method := range supportedMethods {
		methods = append(methods, method)
	}
	return methods
}
