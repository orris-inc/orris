package blockchain

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/orris-inc/orris/internal/application/payment/blockchain"
	vo "github.com/orris-inc/orris/internal/domain/payment/valueobjects"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// CompositeMonitor routes transaction monitoring to the appropriate chain-specific monitor
type CompositeMonitor struct {
	mu             sync.RWMutex // Protects monitor fields for concurrent access
	polygonMonitor *PolygonMonitor
	tronMonitor    *TronMonitor
	logger         logger.Interface
}

// NewCompositeMonitor creates a new composite transaction monitor
func NewCompositeMonitor(polygonMonitor *PolygonMonitor, tronMonitor *TronMonitor, logger logger.Interface) *CompositeMonitor {
	return &CompositeMonitor{
		polygonMonitor: polygonMonitor,
		tronMonitor:    tronMonitor,
		logger:         logger,
	}
}

// Ensure CompositeMonitor implements TransactionMonitor
var _ blockchain.TransactionMonitor = (*CompositeMonitor)(nil)

// FindTransaction searches for a transaction matching the given criteria
// amountRaw is the expected amount in smallest unit (1 USDT = 1000000)
// createdAfter filters transactions to only include those after the payment creation time
func (m *CompositeMonitor) FindTransaction(ctx context.Context, chainType vo.ChainType, toAddress string, amountRaw uint64, createdAfter time.Time) (*blockchain.Transaction, error) {
	m.mu.RLock()
	polygonMonitor := m.polygonMonitor
	tronMonitor := m.tronMonitor
	m.mu.RUnlock()

	switch chainType {
	case vo.ChainTypePOL:
		if polygonMonitor == nil {
			return nil, fmt.Errorf("Polygon monitor not configured")
		}
		return polygonMonitor.FindTransaction(ctx, chainType, toAddress, amountRaw, createdAfter)
	case vo.ChainTypeTRC:
		if tronMonitor == nil {
			return nil, fmt.Errorf("Tron monitor not configured")
		}
		return tronMonitor.FindTransaction(ctx, chainType, toAddress, amountRaw, createdAfter)
	default:
		return nil, fmt.Errorf("unsupported chain type: %s", chainType)
	}
}

// GetConfirmations returns the current number of confirmations for a transaction
func (m *CompositeMonitor) GetConfirmations(ctx context.Context, chainType vo.ChainType, txHash string) (int, error) {
	m.mu.RLock()
	polygonMonitor := m.polygonMonitor
	tronMonitor := m.tronMonitor
	m.mu.RUnlock()

	switch chainType {
	case vo.ChainTypePOL:
		if polygonMonitor == nil {
			return 0, fmt.Errorf("Polygon monitor not configured")
		}
		return polygonMonitor.GetConfirmations(ctx, chainType, txHash)
	case vo.ChainTypeTRC:
		if tronMonitor == nil {
			return 0, fmt.Errorf("Tron monitor not configured")
		}
		return tronMonitor.GetConfirmations(ctx, chainType, txHash)
	default:
		return 0, fmt.Errorf("unsupported chain type: %s", chainType)
	}
}

// UpdatePolygonMonitor updates the Polygon monitor with new configuration
func (m *CompositeMonitor) UpdatePolygonMonitor(monitor *PolygonMonitor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.polygonMonitor = monitor
}

// UpdateTronMonitor updates the Tron monitor with new configuration
func (m *CompositeMonitor) UpdateTronMonitor(monitor *TronMonitor) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tronMonitor = monitor
}
