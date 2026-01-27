package dto

// USDTSettingsResponse represents the USDT settings for API response
type USDTSettingsResponse struct {
	Enabled              SettingWithSource `json:"enabled"`
	POLReceivingAddresses SettingWithSource `json:"pol_receiving_addresses"`
	TRCReceivingAddresses SettingWithSource `json:"trc_receiving_addresses"`
	PolygonScanAPIKey    SettingWithSource `json:"polygonscan_api_key"`
	TronGridAPIKey       SettingWithSource `json:"trongrid_api_key"`
	PaymentTTLMinutes    SettingWithSource `json:"payment_ttl_minutes"`
	POLConfirmations     SettingWithSource `json:"pol_confirmations"`
	TRCConfirmations     SettingWithSource `json:"trc_confirmations"`
}

// UpdateUSDTSettingsRequest represents the request to update USDT settings
type UpdateUSDTSettingsRequest struct {
	Enabled              *bool     `json:"enabled"`
	POLReceivingAddresses *[]string `json:"pol_receiving_addresses"`
	TRCReceivingAddresses *[]string `json:"trc_receiving_addresses"`
	PolygonScanAPIKey    *string   `json:"polygonscan_api_key"`
	TronGridAPIKey       *string   `json:"trongrid_api_key"`
	PaymentTTLMinutes    *int      `json:"payment_ttl_minutes"`
	POLConfirmations     *int      `json:"pol_confirmations"`
	TRCConfirmations     *int      `json:"trc_confirmations"`
}

// USDTSettingKeys maps the DTO field names to database keys
var USDTSettingKeys = map[string]string{
	"enabled":                "enabled",
	"pol_receiving_addresses": "pol_receiving_addresses",
	"trc_receiving_addresses": "trc_receiving_addresses",
	"polygonscan_api_key":    "polygonscan_api_key",
	"trongrid_api_key":       "trongrid_api_key",
	"payment_ttl_minutes":    "payment_ttl_minutes",
	"pol_confirmations":      "pol_confirmations",
	"trc_confirmations":      "trc_confirmations",
}
