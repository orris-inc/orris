package valueobjects

// Example usage patterns for PlanFeatures with traffic and resource limits

// ExampleBasicPlan demonstrates creating a basic plan with 100GB monthly traffic
func ExampleBasicPlan() *PlanFeatures {
	features := []string{
		"basic_support",
		"email_notifications",
	}

	limits := map[string]interface{}{
		LimitKeyTraffic:     uint64(100 * 1024 * 1024 * 1024), // 100GB in bytes
		LimitKeyDeviceCount: 3,                                // 3 concurrent devices
		LimitKeySpeedLimit:  100,                              // 100 Mbps
	}

	return NewPlanFeatures(features, limits)
}

// ExampleStandardPlan demonstrates creating a standard plan with 500GB monthly traffic
func ExampleStandardPlan() *PlanFeatures {
	features := []string{
		"priority_support",
		"email_notifications",
		"advanced_analytics",
	}

	limits := map[string]interface{}{
		LimitKeyTraffic:         uint64(500 * 1024 * 1024 * 1024), // 500GB in bytes
		LimitKeyDeviceCount:     5,                                // 5 concurrent devices
		LimitKeySpeedLimit:      500,                              // 500 Mbps
		LimitKeyConnectionLimit: 100,                              // 100 concurrent connections
		LimitKeyNodeAccess:      []uint{1, 2, 3},                  // Access to node groups 1, 2, 3
	}

	return NewPlanFeatures(features, limits)
}

// ExamplePremiumPlan demonstrates creating a premium plan with unlimited traffic
func ExamplePremiumPlan() *PlanFeatures {
	features := []string{
		"24_7_support",
		"email_notifications",
		"sms_notifications",
		"advanced_analytics",
		"custom_branding",
		"api_access",
	}

	limits := map[string]interface{}{
		LimitKeyTraffic:         uint64(0), // 0 = unlimited
		LimitKeyDeviceCount:     10,        // 10 concurrent devices
		LimitKeySpeedLimit:      1000,      // 1 Gbps
		LimitKeyConnectionLimit: 500,       // 500 concurrent connections
		// Empty node access means all nodes accessible
	}

	return NewPlanFeatures(features, limits)
}

// ExampleDynamicFeatureManagement demonstrates dynamic feature management
func ExampleDynamicFeatureManagement() {
	// Create a new plan with basic features
	pf := NewPlanFeatures([]string{"basic_feature"}, map[string]interface{}{
		LimitKeyTraffic: uint64(100 * 1024 * 1024 * 1024), // 100GB
	})

	// Use typed setters for traffic limits (recommended)
	pf.SetTrafficLimit(200 * 1024 * 1024 * 1024) // Upgrade to 200GB
	pf.SetDeviceLimit(5)                         // Set 5 devices
	pf.SetSpeedLimit(200)                        // Set 200 Mbps
	pf.SetConnectionLimit(50)                    // Set 50 connections

	// Add node access restrictions
	pf.SetNodeAccess([]uint{1, 2, 3, 4, 5})

	// Add additional features
	pf.AddFeature("advanced_feature")
	pf.AddFeature("premium_support")

	// Check if a feature exists
	if pf.HasFeature("advanced_feature") {
		// Feature is available
	}

	// Get traffic limit
	trafficLimit, err := pf.GetTrafficLimit()
	if err != nil {
		// Handle error
	}
	_ = trafficLimit

	// Check if traffic is unlimited
	if pf.IsUnlimitedTraffic() {
		// No traffic restrictions
	}

	// Check if user has exceeded traffic
	usedBytes := uint64(150 * 1024 * 1024 * 1024) // 150GB used
	hasRemaining, err := pf.HasTrafficRemaining(usedBytes)
	if err != nil {
		// Handle error
	}
	if !hasRemaining {
		// User has exceeded traffic limit
	}

	// Get device limit
	deviceLimit, err := pf.GetDeviceLimit()
	if err != nil {
		// Handle error
	}
	_ = deviceLimit

	// Get node access
	nodeGroups, err := pf.GetNodeAccess()
	if err != nil {
		// Handle error
	}
	_ = nodeGroups
}

// ExampleTrafficLimitValidation demonstrates traffic limit validation
func ExampleTrafficLimitValidation() *PlanFeatures {
	pf := NewPlanFeatures(nil, nil)

	// Set 1TB monthly traffic limit
	oneTerabyte := uint64(1024 * 1024 * 1024 * 1024)
	pf.SetTrafficLimit(oneTerabyte)

	// Example usage scenarios:
	// 1. User has used 500GB - should have remaining traffic
	usedHalfTB := uint64(500 * 1024 * 1024 * 1024)
	hasRemaining1, _ := pf.HasTrafficRemaining(usedHalfTB)
	_ = hasRemaining1 // true

	// 2. User has used 1.5TB - should exceed traffic limit
	usedOverLimit := uint64(1536 * 1024 * 1024 * 1024)
	hasRemaining2, _ := pf.HasTrafficRemaining(usedOverLimit)
	_ = hasRemaining2 // false

	return pf
}

// ExampleUnlimitedPlan demonstrates creating a plan with no limits
func ExampleUnlimitedPlan() *PlanFeatures {
	features := []string{
		"enterprise_support",
		"all_features",
	}

	limits := map[string]interface{}{
		LimitKeyTraffic:         uint64(0), // unlimited
		LimitKeyDeviceCount:     0,         // unlimited
		LimitKeySpeedLimit:      0,         // unlimited
		LimitKeyConnectionLimit: 0,         // unlimited
		// No node access restrictions - all nodes accessible
	}

	return NewPlanFeatures(features, limits)
}

// ExampleConversionHelpers demonstrates common traffic unit conversions
const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB
	TB = 1024 * GB
)

func ExampleTrafficConversions() {
	// Convert GB to bytes for storage
	traffic100GB := uint64(100 * GB)
	traffic500GB := uint64(500 * GB)
	traffic1TB := uint64(1 * TB)

	pf1 := NewPlanFeatures(nil, map[string]interface{}{
		LimitKeyTraffic: traffic100GB,
	})

	pf2 := NewPlanFeatures(nil, map[string]interface{}{
		LimitKeyTraffic: traffic500GB,
	})

	pf3 := NewPlanFeatures(nil, map[string]interface{}{
		LimitKeyTraffic: traffic1TB,
	})

	_ = pf1
	_ = pf2
	_ = pf3
}
