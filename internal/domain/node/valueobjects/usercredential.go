package valueobjects

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"

	"github.com/google/uuid"
)

// UsesUUIDCredential reports whether the protocol authenticates users by UUID
// (VLESS, VMess, TUIC) instead of a free-form password.
func (p Protocol) UsesUUIDCredential() bool {
	return p == ProtocolVLESS || p == ProtocolVMess || p == ProtocolTUIC
}

// DeriveUUIDFromCredential maps an arbitrary credential string to a canonical UUID.
//
// It uses UUIDv5 with the nil namespace, which is exactly the mapping sing-box
// and mihomo apply internally when given a non-UUID user id. Emitting the mapped
// value explicitly keeps existing clients working while making the id acceptable
// to strict parsers (Xray, sing-box TUIC, Shadowrocket, ...).
// Returns an empty string for an empty credential.
func DeriveUUIDFromCredential(credential string) string {
	if credential == "" {
		return ""
	}
	return uuid.NewSHA1(uuid.Nil, []byte(credential)).String()
}

// GenerateNodeForwardUUID derives the UUID used for node-to-node forwarding
// on UUID-based protocols (VLESS/VMess/TUIC).
func GenerateNodeForwardUUID(tokenHash string) string {
	if tokenHash == "" {
		return ""
	}

	mac := hmac.New(sha256.New, []byte("uuid-server-credential"))
	mac.Write([]byte(tokenHash))

	return DeriveUUIDFromCredential(fmt.Sprintf("%x", mac.Sum(nil)))
}

// GenerateNodeForwardCredential returns the credential other nodes use to forward
// traffic into a node of the given protocol. For UUID-based protocols the value is
// a canonical UUID. Returns an empty string when forwarding is unsupported.
func GenerateNodeForwardCredential(protocol Protocol, tokenHash string, method string) string {
	switch {
	case protocol.IsShadowsocks():
		return GenerateShadowsocksServerPassword(tokenHash, method)
	case protocol.IsTrojan():
		return GenerateTrojanServerPassword(tokenHash)
	case protocol.IsAnyTLS():
		return GenerateAnyTLSServerPassword(tokenHash)
	case protocol.UsesUUIDCredential():
		return GenerateNodeForwardUUID(tokenHash)
	default:
		return ""
	}
}
