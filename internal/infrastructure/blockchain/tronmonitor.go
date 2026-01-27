package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/orris-inc/orris/internal/application/payment/blockchain"
	vo "github.com/orris-inc/orris/internal/domain/payment/valueobjects"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// TronGrid API base URL
	trongridAPIURL = "https://api.trongrid.io"
	// USDT contract address on Tron (TRC-20)
	tronUSDTContract = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
	// HTTP request timeout
	tronRequestTimeout = 15 * time.Second
)

// trc20Transfer represents a TRC-20 transfer from TronGrid
type trc20Transfer struct {
	TransactionID  string `json:"transaction_id"`
	BlockTimestamp int64  `json:"block_timestamp"`
	From           string `json:"from"`
	To             string `json:"to"`
	Value          string `json:"value"`
	TokenInfo      struct {
		Address  string `json:"address"`
		Decimals int    `json:"decimals"`
	} `json:"token_info"`
}

// trc20Response represents the TronGrid TRC-20 transfer response
type trc20Response struct {
	Data    []trc20Transfer `json:"data"`
	Success bool            `json:"success"`
	Meta    struct {
		At          int64  `json:"at"`
		Fingerprint string `json:"fingerprint"`
	} `json:"meta"`
}

// TronMonitor monitors USDT transactions on Tron network
type TronMonitor struct {
	apiKey     string
	httpClient *http.Client
	logger     logger.Interface
}

// NewTronMonitor creates a new Tron transaction monitor
func NewTronMonitor(apiKey string, logger logger.Interface) *TronMonitor {
	return &TronMonitor{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: tronRequestTimeout,
		},
		logger: logger,
	}
}

// FindTransaction searches for a USDT transaction matching the criteria
// amountRaw is the expected amount in smallest unit (1 USDT = 1000000)
// createdAfter filters transactions to only include those after the payment creation time
func (m *TronMonitor) FindTransaction(ctx context.Context, chainType vo.ChainType, toAddress string, amountRaw uint64, createdAfter time.Time) (*blockchain.Transaction, error) {
	if chainType != vo.ChainTypeTRC {
		return nil, fmt.Errorf("TronMonitor only supports Tron chain")
	}

	// Note: Tron addresses are case-sensitive (Base58Check encoding)

	// Query TRC-20 transfers to the receiving address
	url := fmt.Sprintf("%s/v1/accounts/%s/transactions/trc20?only_to=true&limit=20&contract_address=%s",
		trongridAPIURL, toAddress, tronUSDTContract)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set API key header
	if m.apiKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", m.apiKey)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}
	defer resp.Body.Close()

	var apiResp trc20Response
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Success {
		if m.apiKey == "" {
			return nil, fmt.Errorf("TronGrid API key not configured, please set trongrid_api_key in USDT settings")
		}
		return nil, fmt.Errorf("TronGrid API request failed, possibly rate limited or invalid API key")
	}

	// Find matching transaction
	// Note: Tron addresses use Base58Check encoding and are case-sensitive
	for _, transfer := range apiResp.Data {
		if transfer.To != toAddress {
			continue
		}

		// Parse amount from blockchain (returns raw uint64)
		txAmountRaw, err := strconv.ParseUint(transfer.Value, 10, 64)
		if err != nil {
			m.logger.Warnw("failed to parse transaction amount",
				"tx_hash", transfer.TransactionID,
				"value", transfer.Value,
				"error", err,
			)
			continue
		}

		// Exact integer match - no tolerance needed
		if txAmountRaw == amountRaw {
			txTime := time.UnixMilli(transfer.BlockTimestamp)

			// Time window check: only accept transactions after payment creation
			// This prevents attackers from using pre-existing transactions to claim payments
			// Allow 30 seconds buffer for clock skew between system and blockchain
			timeBuffer := 30 * time.Second
			if !createdAfter.IsZero() && txTime.Before(createdAfter.Add(-timeBuffer)) {
				m.logger.Debugw("skipping transaction before payment creation",
					"tx_hash", transfer.TransactionID,
					"tx_time", txTime,
					"payment_created", createdAfter,
					"buffer", timeBuffer,
				)
				continue
			}

			// Get transaction details for block number and confirmations
			blockNumber, confirmations, err := m.getTransactionDetails(ctx, transfer.TransactionID)
			if err != nil {
				m.logger.Warnw("failed to get transaction details",
					"tx_hash", transfer.TransactionID,
					"error", err,
				)
				continue
			}

			m.logger.Infow("found matching USDT transaction",
				"tx_hash", transfer.TransactionID,
				"amount_raw", txAmountRaw,
				"amount_usdt", blockchain.RawAmountToFloat(txAmountRaw),
				"confirmations", confirmations,
				"tx_time", txTime,
			)

			return &blockchain.Transaction{
				TxHash:        transfer.TransactionID,
				FromAddress:   transfer.From,
				ToAddress:     transfer.To,
				AmountRaw:     txAmountRaw,
				BlockNumber:   blockNumber,
				Confirmations: confirmations,
				Timestamp:     txTime,
			}, nil
		}
	}

	return nil, nil
}

// GetConfirmations returns the current number of confirmations for a transaction
func (m *TronMonitor) GetConfirmations(ctx context.Context, chainType vo.ChainType, txHash string) (int, error) {
	if chainType != vo.ChainTypeTRC {
		return 0, fmt.Errorf("TronMonitor only supports Tron chain")
	}

	_, confirmations, err := m.getTransactionDetails(ctx, txHash)
	if err != nil {
		return 0, err
	}

	return confirmations, nil
}

// getTransactionDetails returns the block number and confirmations for a transaction
func (m *TronMonitor) getTransactionDetails(ctx context.Context, txHash string) (uint64, int, error) {
	url := fmt.Sprintf("%s/v1/transactions/%s", trongridAPIURL, txHash)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create request: %w", err)
	}

	if m.apiKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", m.apiKey)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to fetch transaction: %w", err)
	}
	defer resp.Body.Close()

	var txResp struct {
		Data []struct {
			BlockNumber int64 `json:"blockNumber"`
		} `json:"data"`
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&txResp); err != nil {
		return 0, 0, fmt.Errorf("failed to decode response: %w", err)
	}

	if !txResp.Success || len(txResp.Data) == 0 {
		return 0, 0, nil
	}

	blockNumber := uint64(txResp.Data[0].BlockNumber)

	// Get current block number to calculate confirmations
	currentBlock, err := m.getCurrentBlockNumber(ctx)
	if err != nil {
		return blockNumber, 0, nil
	}

	confirmations := int(currentBlock - blockNumber + 1)
	if confirmations < 0 {
		confirmations = 0
	}

	return blockNumber, confirmations, nil
}

// getCurrentBlockNumber returns the current block number on Tron
func (m *TronMonitor) getCurrentBlockNumber(ctx context.Context) (uint64, error) {
	url := fmt.Sprintf("%s/wallet/getnowblock", trongridAPIURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return 0, err
	}

	if m.apiKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", m.apiKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var blockResp struct {
		BlockHeader struct {
			RawData struct {
				Number int64 `json:"number"`
			} `json:"raw_data"`
		} `json:"block_header"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&blockResp); err != nil {
		return 0, err
	}

	return uint64(blockResp.BlockHeader.RawData.Number), nil
}
