package valueobjects

import (
	"fmt"
	"regexp"
)

// ChainType represents the blockchain type for USDT payments
type ChainType string

const (
	// ChainTypePOL represents the Polygon (Matic) blockchain
	ChainTypePOL ChainType = "pol"
	// ChainTypeTRC represents the Tron TRC-20 blockchain
	ChainTypeTRC ChainType = "trc"
)

// NewChainType creates a new ChainType from string
func NewChainType(chainType string) (ChainType, error) {
	ct := ChainType(chainType)
	if !ct.IsValid() {
		return "", fmt.Errorf("invalid chain type: %s", chainType)
	}
	return ct, nil
}

// IsValid checks if the chain type is valid
func (ct ChainType) IsValid() bool {
	switch ct {
	case ChainTypePOL, ChainTypeTRC:
		return true
	default:
		return false
	}
}

// String returns the string representation of the chain type
func (ct ChainType) String() string {
	return string(ct)
}

// RequiredConfirmations returns the number of block confirmations required
// for a transaction to be considered confirmed on this chain
func (ct ChainType) RequiredConfirmations() int {
	switch ct {
	case ChainTypePOL:
		return 12 // Polygon requires ~12 confirmations for finality
	case ChainTypeTRC:
		return 19 // Tron requires ~19 confirmations for finality
	default:
		return 0
	}
}

// USDTContractAddress returns the USDT contract address for this chain
func (ct ChainType) USDTContractAddress() string {
	switch ct {
	case ChainTypePOL:
		return "0xc2132D05D31c914a87C6611C10748AEb04B58e8F" // USDT on Polygon
	case ChainTypeTRC:
		return "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" // USDT on Tron
	default:
		return ""
	}
}

// Address validation patterns
var (
	// Polygon (EVM) address pattern: 0x followed by 40 hex characters
	polygonAddressPattern = regexp.MustCompile(`^0x[a-fA-F0-9]{40}$`)
	// Tron address pattern: T followed by 33 base58 characters
	// Tron uses Base58Check encoding, starting with 'T'
	tronAddressPattern = regexp.MustCompile(`^T[1-9A-HJ-NP-Za-km-z]{33}$`)
)

// ValidateAddress validates a blockchain address for this chain type
func (ct ChainType) ValidateAddress(address string) error {
	if address == "" {
		return fmt.Errorf("address cannot be empty")
	}

	switch ct {
	case ChainTypePOL:
		if !polygonAddressPattern.MatchString(address) {
			return fmt.Errorf("invalid Polygon address format: must be 0x followed by 40 hex characters")
		}
		return nil
	case ChainTypeTRC:
		if !tronAddressPattern.MatchString(address) {
			return fmt.Errorf("invalid Tron address format: must start with T followed by 33 base58 characters")
		}
		return nil
	default:
		return fmt.Errorf("cannot validate address for unknown chain type: %s", ct)
	}
}

// IsValidAddress returns true if the address is valid for this chain type
func (ct ChainType) IsValidAddress(address string) bool {
	return ct.ValidateAddress(address) == nil
}
