package valueobjects

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeriveUUIDFromCredential(t *testing.T) {
	t.Run("empty credential yields empty string", func(t *testing.T) {
		assert.Equal(t, "", DeriveUUIDFromCredential(""))
	})

	t.Run("matches sing-box and mihomo non-UUID mapping", func(t *testing.T) {
		// Expected value computed with github.com/gofrs/uuid/v5 NewV5(uuid.Nil, s),
		// which sing-box (sing-vmess) and mihomo apply to non-UUID user ids.
		// Keeping this identical lets clients with the old 64-hex id keep working.
		hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		assert.Equal(t, "9e77040f-0eca-515c-abfc-53494dd20237", DeriveUUIDFromCredential(hexKey))
	})

	t.Run("output is a canonical UUID", func(t *testing.T) {
		got := DeriveUUIDFromCredential("any-credential")
		parsed, err := uuid.Parse(got)
		require.NoError(t, err)
		assert.Len(t, got, 36)
		assert.Equal(t, uuid.Version(5), parsed.Version())
	})
}

func TestProtocolUsesUUIDCredential(t *testing.T) {
	assert.True(t, ProtocolVLESS.UsesUUIDCredential())
	assert.True(t, ProtocolVMess.UsesUUIDCredential())
	assert.True(t, ProtocolTUIC.UsesUUIDCredential())
	assert.False(t, ProtocolTrojan.UsesUUIDCredential())
	assert.False(t, ProtocolShadowsocks.UsesUUIDCredential())
	assert.False(t, ProtocolHysteria2.UsesUUIDCredential())
	assert.False(t, ProtocolAnyTLS.UsesUUIDCredential())
}

func TestGenerateNodeForwardCredential(t *testing.T) {
	const tokenHash = "token-hash"

	for _, p := range []Protocol{ProtocolVLESS, ProtocolVMess, ProtocolTUIC} {
		got := GenerateNodeForwardCredential(p, tokenHash, "")
		_, err := uuid.Parse(got)
		require.NoError(t, err, "protocol %s", p)
		assert.Equal(t, GenerateNodeForwardUUID(tokenHash), got)
	}

	assert.Equal(t, GenerateTrojanServerPassword(tokenHash), GenerateNodeForwardCredential(ProtocolTrojan, tokenHash, ""))
	assert.Equal(t, GenerateAnyTLSServerPassword(tokenHash), GenerateNodeForwardCredential(ProtocolAnyTLS, tokenHash, ""))
	assert.Equal(t,
		GenerateShadowsocksServerPassword(tokenHash, MethodAES256GCM),
		GenerateNodeForwardCredential(ProtocolShadowsocks, tokenHash, MethodAES256GCM),
	)
	assert.Equal(t, "", GenerateNodeForwardCredential(ProtocolHysteria2, tokenHash, ""))
	assert.Equal(t, "", GenerateNodeForwardCredential(ProtocolVLESS, "", ""))
}
