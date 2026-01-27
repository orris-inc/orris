package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/orris-inc/orris/internal/application/payment/blockchain"
	vo "github.com/orris-inc/orris/internal/domain/payment/valueobjects"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// Etherscan V2 API base URL (unified for all EVM chains)
	etherscanV2APIURL = "https://api.etherscan.io/v2/api"
	// Polygon chain ID for Etherscan V2 API
	polygonChainID = "137"
	// USDT contract address on Polygon
	polygonUSDTContract = "0xc2132D05D31c914a87C6611C10748AEb04B58e8F"
	// HTTP request timeout
	polygonRequestTimeout = 15 * time.Second
)

// polygonscanResponse represents the PolygonScan API response
type polygonscanResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  any    `json:"result"`
}

// polygonTokenTransfer represents a token transfer from PolygonScan
type polygonTokenTransfer struct {
	BlockNumber   string `json:"blockNumber"`
	TimeStamp     string `json:"timeStamp"`
	Hash          string `json:"hash"`
	From          string `json:"from"`
	To            string `json:"to"`
	Value         string `json:"value"`
	ContractAddr  string `json:"contractAddress"`
	TokenDecimal  string `json:"tokenDecimal"`
	Confirmations string `json:"confirmations"`
}

// PolygonMonitor monitors USDT transactions on Polygon network
type PolygonMonitor struct {
	apiKey     string
	httpClient *http.Client
	logger     logger.Interface
}

// NewPolygonMonitor creates a new Polygon transaction monitor
func NewPolygonMonitor(apiKey string, logger logger.Interface) *PolygonMonitor {
	return &PolygonMonitor{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: polygonRequestTimeout,
		},
		logger: logger,
	}
}

// FindTransaction searches for a USDT transaction matching the criteria
// amountRaw is the expected amount in smallest unit (1 USDT = 1000000)
// createdAfter filters transactions to only include those after the payment creation time
func (m *PolygonMonitor) FindTransaction(ctx context.Context, chainType vo.ChainType, toAddress string, amountRaw uint64, createdAfter time.Time) (*blockchain.Transaction, error) {
	if chainType != vo.ChainTypePOL {
		return nil, fmt.Errorf("PolygonMonitor only supports Polygon chain")
	}

	// Skip if API key not configured (no error, just return nil)
	if m.apiKey == "" {
		m.logger.Warnw("skipping Polygon transaction check, Etherscan API key not configured")
		return nil, nil
	}

	// Normalize address to lowercase
	toAddress = strings.ToLower(toAddress)

	// Query token transfers to the receiving address (Etherscan V2 API)
	url := fmt.Sprintf("%s?chainid=%s&module=account&action=tokentx&contractaddress=%s&address=%s&page=1&offset=20&sort=desc&apikey=%s",
		etherscanV2APIURL, polygonChainID, polygonUSDTContract, toAddress, m.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}
	defer resp.Body.Close()

	var apiResp polygonscanResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.Status != "1" {
		// No transactions found or API error
		if apiResp.Message == "No transactions found" {
			return nil, nil
		}
		// NOTOK typically means rate limited
		if apiResp.Message == "NOTOK" {
			// Try to get more details from result
			if resultStr, ok := apiResp.Result.(string); ok && resultStr != "" {
				return nil, fmt.Errorf("Etherscan API error: %s", resultStr)
			}
			return nil, fmt.Errorf("Etherscan API rate limited, please try again later")
		}
		return nil, fmt.Errorf("Etherscan API error: %s", apiResp.Message)
	}

	// Parse the result as array of transfers
	resultBytes, err := json.Marshal(apiResp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var transfers []polygonTokenTransfer
	if err := json.Unmarshal(resultBytes, &transfers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transfers: %w", err)
	}

	// Find matching transaction
	for _, transfer := range transfers {
		if strings.ToLower(transfer.To) != toAddress {
			continue
		}

		// Parse amount from blockchain (returns raw uint64)
		txAmountRaw, err := parseUSDTAmountRaw(transfer.Value)
		if err != nil {
			m.logger.Warnw("failed to parse transaction amount",
				"tx_hash", transfer.Hash,
				"value", transfer.Value,
				"error", err,
			)
			continue
		}

		// Exact integer match - no tolerance needed
		if txAmountRaw == amountRaw {
			blockNumber, _ := strconv.ParseUint(transfer.BlockNumber, 10, 64)
			confirmations, _ := strconv.Atoi(transfer.Confirmations)
			timestamp, _ := strconv.ParseInt(transfer.TimeStamp, 10, 64)
			txTime := time.Unix(timestamp, 0)

			// Time window check: only accept transactions after payment creation
			// This prevents attackers from using pre-existing transactions to claim payments
			// Allow 30 seconds buffer for clock skew between system and blockchain
			timeBuffer := 30 * time.Second
			if !createdAfter.IsZero() && txTime.Before(createdAfter.Add(-timeBuffer)) {
				m.logger.Debugw("skipping transaction before payment creation",
					"tx_hash", transfer.Hash,
					"tx_time", txTime,
					"payment_created", createdAfter,
					"buffer", timeBuffer,
				)
				continue
			}

			m.logger.Infow("found matching USDT transaction",
				"tx_hash", transfer.Hash,
				"amount_raw", txAmountRaw,
				"amount_usdt", blockchain.RawAmountToFloat(txAmountRaw),
				"confirmations", confirmations,
				"tx_time", txTime,
			)

			return &blockchain.Transaction{
				TxHash:        transfer.Hash,
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
func (m *PolygonMonitor) GetConfirmations(ctx context.Context, chainType vo.ChainType, txHash string) (int, error) {
	if chainType != vo.ChainTypePOL {
		return 0, fmt.Errorf("PolygonMonitor only supports Polygon chain")
	}

	// Skip if API key not configured
	if m.apiKey == "" {
		return 0, nil
	}

	url := fmt.Sprintf("%s?chainid=%s&module=proxy&action=eth_getTransactionReceipt&txhash=%s&apikey=%s",
		etherscanV2APIURL, polygonChainID, txHash, m.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch transaction: %w", err)
	}
	defer resp.Body.Close()

	var apiResp struct {
		Result struct {
			BlockNumber string `json:"blockNumber"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.Result.BlockNumber == "" {
		return 0, nil // Transaction not yet confirmed
	}

	txBlockNumber, _ := strconv.ParseInt(strings.TrimPrefix(apiResp.Result.BlockNumber, "0x"), 16, 64)

	// Get current block number
	currentBlock, err := m.getCurrentBlockNumber(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get current block: %w", err)
	}

	confirmations := currentBlock - txBlockNumber + 1
	if confirmations < 0 {
		confirmations = 0
	}

	return int(confirmations), nil
}

// getCurrentBlockNumber returns the current block number on Polygon
func (m *PolygonMonitor) getCurrentBlockNumber(ctx context.Context) (int64, error) {
	// Skip if API key not configured
	if m.apiKey == "" {
		return 0, fmt.Errorf("API key not configured")
	}

	url := fmt.Sprintf("%s?chainid=%s&module=proxy&action=eth_blockNumber&apikey=%s",
		etherscanV2APIURL, polygonChainID, m.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return 0, err
	}

	blockNumber, _ := strconv.ParseInt(strings.TrimPrefix(apiResp.Result, "0x"), 16, 64)
	return blockNumber, nil
}

// parseUSDTAmountRaw parses a USDT amount string to raw uint64 (smallest unit)
// The value from blockchain API is already in smallest unit (e.g., "10123400" for 10.1234 USDT)
func parseUSDTAmountRaw(value string) (uint64, error) {
	amount, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount: %s", value)
	}
	return amount, nil
}
