package valueobjects

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const (
	MethodAES256GCM             = "aes-256-gcm"
	MethodAES128GCM             = "aes-128-gcm"
	MethodAES128CFB             = "aes-128-cfb"
	MethodAES192CFB             = "aes-192-cfb"
	MethodAES256CFB             = "aes-256-cfb"
	MethodAES128CTR             = "aes-128-ctr"
	MethodAES192CTR             = "aes-192-ctr"
	MethodAES256CTR             = "aes-256-ctr"
	MethodChacha20IETF          = "chacha20-ietf"
	MethodChacha20IETFPoly1305  = "chacha20-ietf-poly1305"
	MethodXChacha20IETFPoly1305 = "xchacha20-ietf-poly1305"
	MethodRC4MD5                = "rc4-md5"

	// SS2022 encryption methods
	Method2022Blake3AES128GCM        = "2022-blake3-aes-128-gcm"
	Method2022Blake3AES256GCM        = "2022-blake3-aes-256-gcm"
	Method2022Blake3Chacha20Poly1305 = "2022-blake3-chacha20-poly1305"
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

	// SS2022 methods
	Method2022Blake3AES128GCM:        true,
	Method2022Blake3AES256GCM:        true,
	Method2022Blake3Chacha20Poly1305: true,
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

// IsSS2022Method checks if the encryption method is a SS2022 cipher
func IsSS2022Method(method string) bool {
	switch method {
	case Method2022Blake3AES128GCM,
		Method2022Blake3AES256GCM,
		Method2022Blake3Chacha20Poly1305:
		return true
	default:
		return false
	}
}

// GetSS2022KeySize returns the required key size in bytes for SS2022 methods
// Returns 0 for non-SS2022 methods
func GetSS2022KeySize(method string) int {
	switch method {
	case Method2022Blake3AES128GCM:
		return 16
	case Method2022Blake3AES256GCM,
		Method2022Blake3Chacha20Poly1305:
		return 32
	default:
		return 0
	}
}

// GenerateSS2022ServerKey derives a server PSK from node token hash using HMAC-SHA256.
// The returned key is base64-encoded with proper length for the encryption method.
// For non-SS2022 methods, returns empty string.
func GenerateSS2022ServerKey(tokenHash string, method string) string {
	if tokenHash == "" || !IsSS2022Method(method) {
		return ""
	}

	keySize := GetSS2022KeySize(method)
	if keySize == 0 {
		return ""
	}

	// Derive key material using HMAC-SHA256 with a fixed salt
	mac := hmac.New(sha256.New, []byte("ss2022-server-key"))
	mac.Write([]byte(tokenHash))
	keyMaterial := mac.Sum(nil) // 32 bytes

	// Truncate to required key size and encode as base64
	return base64.StdEncoding.EncodeToString(keyMaterial[:keySize])
}

// GenerateShadowsocksServerPassword derives a Shadowsocks password from node token hash.
// For SS2022 methods, returns base64-encoded key with proper length.
// For traditional SS methods, returns hex-encoded 32-byte password.
// This is used for node-to-node forwarding (outbound) scenarios.
func GenerateShadowsocksServerPassword(tokenHash string, method string) string {
	if tokenHash == "" {
		return ""
	}

	// For SS2022 methods, use base64-encoded key
	if IsSS2022Method(method) {
		return GenerateSS2022ServerKey(tokenHash, method)
	}

	// For traditional SS methods, use hex-encoded password
	mac := hmac.New(sha256.New, []byte("ss-server-password"))
	mac.Write([]byte(tokenHash))
	keyMaterial := mac.Sum(nil) // 32 bytes

	return fmt.Sprintf("%x", keyMaterial)
}

// GenerateTrojanServerPassword derives a Trojan password from node token hash.
// This is used for node-to-node forwarding (outbound) scenarios.
// Returns a hex-encoded 32-byte password derived using HMAC-SHA256.
func GenerateTrojanServerPassword(tokenHash string) string {
	if tokenHash == "" {
		return ""
	}

	// Derive password using HMAC-SHA256 with a fixed salt
	mac := hmac.New(sha256.New, []byte("trojan-server-password"))
	mac.Write([]byte(tokenHash))
	keyMaterial := mac.Sum(nil) // 32 bytes

	// Return as hex string (64 chars, common Trojan password format)
	return fmt.Sprintf("%x", keyMaterial)
}
