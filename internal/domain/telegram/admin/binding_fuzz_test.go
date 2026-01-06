package admin

import (
	"math"
	"testing"
	"unicode/utf8"
)

// FuzzNewAdminTelegramBinding tests NewAdminTelegramBinding with random inputs
func FuzzNewAdminTelegramBinding(f *testing.F) {
	seeds := []struct {
		userID           uint
		telegramUserID   int64
		telegramUsername string
	}{
		{1, 12345, "testuser"},
		{0, 12345, "testuser"},    // Invalid: zero userID
		{1, 0, "testuser"},        // Invalid: zero telegramUserID
		{1, 12345, ""},            // Valid: empty username is OK
		{math.MaxUint32, math.MaxInt64, "maxuser"},
		{1, -12345, "negative"},   // Negative telegramUserID (could be valid)
		{1, 12345, "中文用户"},
		{1, 12345, "@username"},
	}

	for _, seed := range seeds {
		f.Add(seed.userID, seed.telegramUserID, seed.telegramUsername)
	}

	f.Fuzz(func(t *testing.T, userID uint, telegramUserID int64, telegramUsername string) {
		if !utf8.ValidString(telegramUsername) {
			return
		}

		binding, err := NewAdminTelegramBinding(userID, telegramUserID, telegramUsername)

		// Zero userID should error
		if userID == 0 {
			if err == nil {
				t.Errorf("NewAdminTelegramBinding(%d, %d, %q) should error for zero userID", userID, telegramUserID, telegramUsername)
			}
			return
		}

		// Zero telegramUserID should error
		if telegramUserID == 0 {
			if err == nil {
				t.Errorf("NewAdminTelegramBinding(%d, %d, %q) should error for zero telegramUserID", userID, telegramUserID, telegramUsername)
			}
			return
		}

		// Valid inputs should succeed
		if err != nil {
			t.Errorf("NewAdminTelegramBinding(%d, %d, %q) returned unexpected error: %v", userID, telegramUserID, telegramUsername, err)
			return
		}

		// Verify binding was created correctly
		if binding.UserID() != userID {
			t.Errorf("UserID() = %d, expected %d", binding.UserID(), userID)
		}
		if binding.TelegramUserID() != telegramUserID {
			t.Errorf("TelegramUserID() = %d, expected %d", binding.TelegramUserID(), telegramUserID)
		}
		if binding.TelegramUsername() != telegramUsername {
			t.Errorf("TelegramUsername() = %q, expected %q", binding.TelegramUsername(), telegramUsername)
		}

		// Verify defaults
		if !binding.NotifyNodeOffline() {
			t.Error("NotifyNodeOffline should default to true")
		}
		if !binding.NotifyAgentOffline() {
			t.Error("NotifyAgentOffline should default to true")
		}
		if binding.OfflineThresholdMinutes() != DefaultOfflineThresholdMinutes {
			t.Errorf("OfflineThresholdMinutes() = %d, expected %d", binding.OfflineThresholdMinutes(), DefaultOfflineThresholdMinutes)
		}
	})
}

// FuzzUpdatePreferences tests UpdatePreferences with random threshold values
func FuzzUpdatePreferences(f *testing.F) {
	thresholds := []int{
		0, 1, 2, 3, 4, 5, 10, 15, 20, 25, 30, 31, 50, 100, -1, -10,
		MinOfflineThresholdMinutes,
		MaxOfflineThresholdMinutes,
		MinOfflineThresholdMinutes - 1,
		MaxOfflineThresholdMinutes + 1,
	}

	for _, threshold := range thresholds {
		f.Add(threshold)
	}

	f.Fuzz(func(t *testing.T, threshold int) {
		binding, err := NewAdminTelegramBinding(1, 12345, "test")
		if err != nil {
			t.Fatalf("Failed to create binding: %v", err)
		}

		err = binding.UpdatePreferences(nil, nil, nil, nil, nil, nil, &threshold)

		// Threshold out of range should error
		if threshold < MinOfflineThresholdMinutes || threshold > MaxOfflineThresholdMinutes {
			if err == nil {
				t.Errorf("UpdatePreferences with threshold=%d should error (range: %d-%d)", threshold, MinOfflineThresholdMinutes, MaxOfflineThresholdMinutes)
			}
			return
		}

		// Valid threshold should succeed
		if err != nil {
			t.Errorf("UpdatePreferences with threshold=%d returned unexpected error: %v", threshold, err)
			return
		}

		if binding.OfflineThresholdMinutes() != threshold {
			t.Errorf("OfflineThresholdMinutes() = %d, expected %d", binding.OfflineThresholdMinutes(), threshold)
		}
	})
}

// FuzzUpdatePreferencesBooleans tests UpdatePreferences with boolean combinations
func FuzzUpdatePreferencesBooleans(f *testing.F) {
	// Add seeds for all boolean combinations
	for _, nodeOffline := range []bool{true, false} {
		for _, agentOffline := range []bool{true, false} {
			for _, newUser := range []bool{true, false} {
				for _, paymentSuccess := range []bool{true, false} {
					for _, dailySummary := range []bool{true, false} {
						for _, weeklySummary := range []bool{true, false} {
							f.Add(nodeOffline, agentOffline, newUser, paymentSuccess, dailySummary, weeklySummary)
						}
					}
				}
			}
		}
	}

	f.Fuzz(func(t *testing.T, nodeOffline, agentOffline, newUser, paymentSuccess, dailySummary, weeklySummary bool) {
		binding, err := NewAdminTelegramBinding(1, 12345, "test")
		if err != nil {
			t.Fatalf("Failed to create binding: %v", err)
		}

		err = binding.UpdatePreferences(
			&nodeOffline,
			&agentOffline,
			&newUser,
			&paymentSuccess,
			&dailySummary,
			&weeklySummary,
			nil,
		)

		if err != nil {
			t.Errorf("UpdatePreferences returned unexpected error: %v", err)
			return
		}

		// Verify all preferences were set correctly
		if binding.NotifyNodeOffline() != nodeOffline {
			t.Errorf("NotifyNodeOffline() = %t, expected %t", binding.NotifyNodeOffline(), nodeOffline)
		}
		if binding.NotifyAgentOffline() != agentOffline {
			t.Errorf("NotifyAgentOffline() = %t, expected %t", binding.NotifyAgentOffline(), agentOffline)
		}
		if binding.NotifyNewUser() != newUser {
			t.Errorf("NotifyNewUser() = %t, expected %t", binding.NotifyNewUser(), newUser)
		}
		if binding.NotifyPaymentSuccess() != paymentSuccess {
			t.Errorf("NotifyPaymentSuccess() = %t, expected %t", binding.NotifyPaymentSuccess(), paymentSuccess)
		}
		if binding.NotifyDailySummary() != dailySummary {
			t.Errorf("NotifyDailySummary() = %t, expected %t", binding.NotifyDailySummary(), dailySummary)
		}
		if binding.NotifyWeeklySummary() != weeklySummary {
			t.Errorf("NotifyWeeklySummary() = %t, expected %t", binding.NotifyWeeklySummary(), weeklySummary)
		}
	})
}

// FuzzUpdateTelegramUsername tests UpdateTelegramUsername with random inputs
func FuzzUpdateTelegramUsername(f *testing.F) {
	seeds := []string{
		"",
		"testuser",
		"@testuser",
		"user_123",
		"中文用户",
		"emoji👍user",
		"very_long_username_that_exceeds_normal_limits_for_telegram",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, username string) {
		if !utf8.ValidString(username) {
			return
		}

		binding, err := NewAdminTelegramBinding(1, 12345, "original")
		if err != nil {
			t.Fatalf("Failed to create binding: %v", err)
		}

		binding.UpdateTelegramUsername(username)

		if binding.TelegramUsername() != username {
			t.Errorf("TelegramUsername() = %q, expected %q", binding.TelegramUsername(), username)
		}
	})
}
