package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	// Maximum response body size for blockchain API (1MB)
	maxBlockchainResponseSize = 1 << 20
	// Maximum pages to scan to prevent DoS
	maxPolygonPages = 5
	// Results per page
	polygonPageSize = 200
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

	// Allow 30 seconds buffer for clock skew between system and blockchain
	timeBuffer := 30 * time.Second
	minTime := createdAfter.Add(-timeBuffer)

	// Scan multiple pages with early termination when transactions become too old
	for page := 1; page <= maxPolygonPages; page++ {
		tx, shouldStop, err := m.scanPage(ctx, toAddress, amountRaw, minTime, page)
		if err != nil {
			return nil, err
		}
		if tx != nil {
			return tx, nil
		}
		if shouldStop {
			// All transactions on this page are older than our time window
			break
		}
	}

	return nil, nil
}

// scanPage fetches a single page of transactions and searches for a match
// Returns (transaction, shouldStop, error) where shouldStop indicates if we've gone past the time window
func (m *PolygonMonitor) scanPage(ctx context.Context, toAddress string, amountRaw uint64, minTime time.Time, page int) (*blockchain.Transaction, bool, error) {
	transfers, err := m.fetchTokenTransfers(ctx, toAddress, page)
	if err != nil {
		return nil, false, err
	}

	if len(transfers) == 0 {
		return nil, true, nil // No more transactions
	}

	// Find matching transaction
	for _, transfer := range transfers {
		if strings.ToLower(transfer.To) != toAddress {
			continue
		}

		// Parse timestamp first for early termination check
		timestamp, _ := strconv.ParseInt(transfer.TimeStamp, 10, 64)
		txTime := time.Unix(timestamp, 0)

		// Early termination: since results are sorted desc, if we see a tx older than minTime,
		// all subsequent transactions will also be older
		if !minTime.IsZero() && txTime.Before(minTime) {
			m.logger.Debugw("stopping scan: transaction older than payment creation",
				"tx_hash", transfer.Hash,
				"tx_time", txTime,
				"min_time", minTime,
			)
			return nil, true, nil
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
			}, false, nil
		}
	}

	return nil, false, nil
}

// fetchTokenTransfers fetches a page of token transfers from Etherscan API
func (m *PolygonMonitor) fetchTokenTransfers(ctx context.Context, toAddress string, page int) ([]polygonTokenTransfer, error) {
	url := fmt.Sprintf("%s?chainid=%s&module=account&action=tokentx&contractaddress=%s&address=%s&page=%d&offset=%d&sort=desc&apikey=%s",
		etherscanV2APIURL, polygonChainID, polygonUSDTContract, toAddress, page, polygonPageSize, m.apiKey)

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
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBlockchainResponseSize)).Decode(&apiResp); err != nil {
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

	return transfers, nil
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
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBlockchainResponseSize)).Decode(&apiResp); err != nil {
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
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBlockchainResponseSize)).Decode(&apiResp); err != nil {
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
