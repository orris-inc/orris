package subscription

import (
	"testing"
	"time"
)

func TestNewSubscriptionUsage(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		subscriptionID uint
		period         time.Time
		wantErr        bool
		errMsg         string
	}{
		{
			name:           "valid usage",
			subscriptionID: 1,
			period:         now,
			wantErr:        false,
		},
		{
			name:           "zero subscription ID",
			subscriptionID: 0,
			period:         now,
			wantErr:        true,
			errMsg:         "subscription ID cannot be zero",
		},
		{
			name:           "zero period",
			subscriptionID: 1,
			period:         time.Time{},
			wantErr:        true,
			errMsg:         "period cannot be zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage, err := NewSubscriptionUsage(tt.subscriptionID, tt.period)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewSubscriptionUsage() expected error, got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("NewSubscriptionUsage() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("NewSubscriptionUsage() unexpected error = %v", err)
				return
			}

			if usage.subscriptionID != tt.subscriptionID {
				t.Errorf("subscriptionID = %v, want %v", usage.subscriptionID, tt.subscriptionID)
			}

			if !usage.period.Equal(tt.period) {
				t.Errorf("period = %v, want %v", usage.period, tt.period)
			}

			if usage.apiRequests != 0 {
				t.Errorf("apiRequests = %v, want 0", usage.apiRequests)
			}

			if usage.apiDataOut != 0 {
				t.Errorf("apiDataOut = %v, want 0", usage.apiDataOut)
			}

			if usage.updatedAt.IsZero() {
				t.Errorf("updatedAt should be set")
			}
		})
	}
}

func TestReconstructSubscriptionUsage(t *testing.T) {
	now := time.Now()
	period := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name             string
		id               uint
		subscriptionID   uint
		period           time.Time
		apiRequests      uint64
		apiDataOut       uint64
		apiDataIn        uint64
		storageUsed      uint64
		usersCount       uint
		projectsCount    uint
		webhookCalls     uint64
		emailsSent       uint64
		reportsGenerated uint
		updatedAt        time.Time
		wantErr          bool
	}{
		{
			name:             "valid reconstruction",
			id:               1,
			subscriptionID:   100,
			period:           period,
			apiRequests:      1000,
			apiDataOut:       5000000,
			apiDataIn:        3000000,
			storageUsed:      10000000,
			usersCount:       5,
			projectsCount:    3,
			webhookCalls:     200,
			emailsSent:       50,
			reportsGenerated: 10,
			updatedAt:        now,
			wantErr:          false,
		},
		{
			name:             "zero ID",
			id:               0,
			subscriptionID:   100,
			period:           period,
			apiRequests:      0,
			apiDataOut:       0,
			apiDataIn:        0,
			storageUsed:      0,
			usersCount:       0,
			projectsCount:    0,
			webhookCalls:     0,
			emailsSent:       0,
			reportsGenerated: 0,
			updatedAt:        now,
			wantErr:          true,
		},
		{
			name:             "zero subscription ID",
			id:               1,
			subscriptionID:   0,
			period:           period,
			apiRequests:      0,
			apiDataOut:       0,
			apiDataIn:        0,
			storageUsed:      0,
			usersCount:       0,
			projectsCount:    0,
			webhookCalls:     0,
			emailsSent:       0,
			reportsGenerated: 0,
			updatedAt:        now,
			wantErr:          true,
		},
		{
			name:             "zero period",
			id:               1,
			subscriptionID:   100,
			period:           time.Time{},
			apiRequests:      0,
			apiDataOut:       0,
			apiDataIn:        0,
			storageUsed:      0,
			usersCount:       0,
			projectsCount:    0,
			webhookCalls:     0,
			emailsSent:       0,
			reportsGenerated: 0,
			updatedAt:        now,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage, err := ReconstructSubscriptionUsage(
				tt.id,
				tt.subscriptionID,
				tt.period,
				tt.apiRequests,
				tt.apiDataOut,
				tt.apiDataIn,
				tt.storageUsed,
				tt.usersCount,
				tt.projectsCount,
				tt.webhookCalls,
				tt.emailsSent,
				tt.reportsGenerated,
				tt.updatedAt,
			)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ReconstructSubscriptionUsage() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ReconstructSubscriptionUsage() unexpected error = %v", err)
				return
			}

			if usage.id != tt.id {
				t.Errorf("id = %v, want %v", usage.id, tt.id)
			}

			if usage.apiRequests != tt.apiRequests {
				t.Errorf("apiRequests = %v, want %v", usage.apiRequests, tt.apiRequests)
			}
		})
	}
}

func TestSubscriptionUsage_IncrementMethods(t *testing.T) {
	now := time.Now()
	usage, _ := NewSubscriptionUsage(1, now)

	t.Run("IncrementAPIRequests", func(t *testing.T) {
		initial := usage.apiRequests
		usage.IncrementAPIRequests(100)

		if usage.apiRequests != initial+100 {
			t.Errorf("apiRequests = %v, want %v", usage.apiRequests, initial+100)
		}
	})

	t.Run("IncrementAPIDataOut", func(t *testing.T) {
		initial := usage.apiDataOut
		usage.IncrementAPIDataOut(5000)

		if usage.apiDataOut != initial+5000 {
			t.Errorf("apiDataOut = %v, want %v", usage.apiDataOut, initial+5000)
		}
	})

	t.Run("IncrementAPIDataIn", func(t *testing.T) {
		initial := usage.apiDataIn
		usage.IncrementAPIDataIn(3000)

		if usage.apiDataIn != initial+3000 {
			t.Errorf("apiDataIn = %v, want %v", usage.apiDataIn, initial+3000)
		}
	})

	t.Run("IncrementStorageUsed", func(t *testing.T) {
		initial := usage.storageUsed
		usage.IncrementStorageUsed(10000)

		if usage.storageUsed != initial+10000 {
			t.Errorf("storageUsed = %v, want %v", usage.storageUsed, initial+10000)
		}
	})

	t.Run("IncrementWebhookCalls", func(t *testing.T) {
		initial := usage.webhookCalls
		usage.IncrementWebhookCalls(50)

		if usage.webhookCalls != initial+50 {
			t.Errorf("webhookCalls = %v, want %v", usage.webhookCalls, initial+50)
		}
	})

	t.Run("IncrementEmailsSent", func(t *testing.T) {
		initial := usage.emailsSent
		usage.IncrementEmailsSent(25)

		if usage.emailsSent != initial+25 {
			t.Errorf("emailsSent = %v, want %v", usage.emailsSent, initial+25)
		}
	})

	t.Run("IncrementReportsGenerated", func(t *testing.T) {
		initial := usage.reportsGenerated
		usage.IncrementReportsGenerated()

		if usage.reportsGenerated != initial+1 {
			t.Errorf("reportsGenerated = %v, want %v", usage.reportsGenerated, initial+1)
		}
	})
}

func TestSubscriptionUsage_CounterMethods(t *testing.T) {
	now := time.Now()
	usage, _ := NewSubscriptionUsage(1, now)

	t.Run("IncrementUsersCount", func(t *testing.T) {
		initial := usage.usersCount
		usage.IncrementUsersCount()

		if usage.usersCount != initial+1 {
			t.Errorf("usersCount = %v, want %v", usage.usersCount, initial+1)
		}
	})

	t.Run("DecrementUsersCount", func(t *testing.T) {
		usage.IncrementUsersCount()
		usage.IncrementUsersCount()

		initial := usage.usersCount
		usage.DecrementUsersCount()

		if usage.usersCount != initial-1 {
			t.Errorf("usersCount = %v, want %v", usage.usersCount, initial-1)
		}
	})

	t.Run("DecrementUsersCount at zero", func(t *testing.T) {
		usage2, _ := NewSubscriptionUsage(1, now)
		usage2.DecrementUsersCount()

		if usage2.usersCount != 0 {
			t.Errorf("usersCount should remain 0")
		}
	})

	t.Run("IncrementProjectsCount", func(t *testing.T) {
		initial := usage.projectsCount
		usage.IncrementProjectsCount()

		if usage.projectsCount != initial+1 {
			t.Errorf("projectsCount = %v, want %v", usage.projectsCount, initial+1)
		}
	})

	t.Run("DecrementProjectsCount", func(t *testing.T) {
		usage.IncrementProjectsCount()
		usage.IncrementProjectsCount()

		initial := usage.projectsCount
		usage.DecrementProjectsCount()

		if usage.projectsCount != initial-1 {
			t.Errorf("projectsCount = %v, want %v", usage.projectsCount, initial-1)
		}
	})

	t.Run("DecrementProjectsCount at zero", func(t *testing.T) {
		usage3, _ := NewSubscriptionUsage(1, now)
		usage3.DecrementProjectsCount()

		if usage3.projectsCount != 0 {
			t.Errorf("projectsCount should remain 0")
		}
	})
}

func TestSubscriptionUsage_DecrementStorageUsed(t *testing.T) {
	now := time.Now()
	usage, _ := NewSubscriptionUsage(1, now)

	usage.IncrementStorageUsed(10000)

	t.Run("normal decrement", func(t *testing.T) {
		initial := usage.storageUsed
		usage.DecrementStorageUsed(3000)

		if usage.storageUsed != initial-3000 {
			t.Errorf("storageUsed = %v, want %v", usage.storageUsed, initial-3000)
		}
	})

	t.Run("decrement more than available", func(t *testing.T) {
		usage.DecrementStorageUsed(100000)

		if usage.storageUsed != 0 {
			t.Errorf("storageUsed = %v, want 0", usage.storageUsed)
		}
	})
}

func TestSubscriptionUsage_Reset(t *testing.T) {
	now := time.Now()
	usage, _ := NewSubscriptionUsage(1, now)

	usage.IncrementAPIRequests(100)
	usage.IncrementAPIDataOut(5000)
	usage.IncrementAPIDataIn(3000)
	usage.IncrementStorageUsed(10000)
	usage.IncrementUsersCount()
	usage.IncrementProjectsCount()
	usage.IncrementWebhookCalls(50)
	usage.IncrementEmailsSent(25)
	usage.IncrementReportsGenerated()

	usage.Reset()

	if usage.apiRequests != 0 {
		t.Errorf("apiRequests = %v, want 0", usage.apiRequests)
	}

	if usage.apiDataOut != 0 {
		t.Errorf("apiDataOut = %v, want 0", usage.apiDataOut)
	}

	if usage.apiDataIn != 0 {
		t.Errorf("apiDataIn = %v, want 0", usage.apiDataIn)
	}

	if usage.storageUsed != 0 {
		t.Errorf("storageUsed = %v, want 0", usage.storageUsed)
	}

	if usage.usersCount != 0 {
		t.Errorf("usersCount = %v, want 0", usage.usersCount)
	}

	if usage.projectsCount != 0 {
		t.Errorf("projectsCount = %v, want 0", usage.projectsCount)
	}

	if usage.webhookCalls != 0 {
		t.Errorf("webhookCalls = %v, want 0", usage.webhookCalls)
	}

	if usage.emailsSent != 0 {
		t.Errorf("emailsSent = %v, want 0", usage.emailsSent)
	}

	if usage.reportsGenerated != 0 {
		t.Errorf("reportsGenerated = %v, want 0", usage.reportsGenerated)
	}
}

func TestSubscriptionUsage_Getters(t *testing.T) {
	now := time.Now()
	period := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	usage, _ := ReconstructSubscriptionUsage(
		1,
		100,
		period,
		1000,
		5000000,
		3000000,
		10000000,
		5,
		3,
		200,
		50,
		10,
		now,
	)

	if usage.ID() != 1 {
		t.Errorf("ID() = %v, want 1", usage.ID())
	}

	if usage.SubscriptionID() != 100 {
		t.Errorf("SubscriptionID() = %v, want 100", usage.SubscriptionID())
	}

	if !usage.Period().Equal(period) {
		t.Errorf("Period() = %v, want %v", usage.Period(), period)
	}

	if usage.APIRequests() != 1000 {
		t.Errorf("APIRequests() = %v, want 1000", usage.APIRequests())
	}

	if usage.APIDataOut() != 5000000 {
		t.Errorf("APIDataOut() = %v, want 5000000", usage.APIDataOut())
	}

	if usage.APIDataIn() != 3000000 {
		t.Errorf("APIDataIn() = %v, want 3000000", usage.APIDataIn())
	}

	if usage.StorageUsed() != 10000000 {
		t.Errorf("StorageUsed() = %v, want 10000000", usage.StorageUsed())
	}

	if usage.UsersCount() != 5 {
		t.Errorf("UsersCount() = %v, want 5", usage.UsersCount())
	}

	if usage.ProjectsCount() != 3 {
		t.Errorf("ProjectsCount() = %v, want 3", usage.ProjectsCount())
	}

	if usage.WebhookCalls() != 200 {
		t.Errorf("WebhookCalls() = %v, want 200", usage.WebhookCalls())
	}

	if usage.EmailsSent() != 50 {
		t.Errorf("EmailsSent() = %v, want 50", usage.EmailsSent())
	}

	if usage.ReportsGenerated() != 10 {
		t.Errorf("ReportsGenerated() = %v, want 10", usage.ReportsGenerated())
	}
}

func TestSubscriptionUsage_GetTotalAPIData(t *testing.T) {
	now := time.Now()
	usage, _ := NewSubscriptionUsage(1, now)

	usage.IncrementAPIDataOut(5000000)
	usage.IncrementAPIDataIn(3000000)

	total := usage.GetTotalAPIData()

	if total != 8000000 {
		t.Errorf("GetTotalAPIData() = %v, want 8000000", total)
	}
}

func TestSubscriptionUsage_GetTotalActivity(t *testing.T) {
	now := time.Now()
	usage, _ := NewSubscriptionUsage(1, now)

	usage.IncrementAPIRequests(100)
	usage.IncrementWebhookCalls(50)
	usage.IncrementEmailsSent(25)
	usage.IncrementReportsGenerated()
	usage.IncrementReportsGenerated()

	total := usage.GetTotalActivity()

	if total != 177 {
		t.Errorf("GetTotalActivity() = %v, want 177", total)
	}
}

func TestSubscriptionUsage_HasUsage(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name   string
		setup  func(*SubscriptionUsage)
		want   bool
	}{
		{
			name:   "no usage",
			setup:  func(u *SubscriptionUsage) {},
			want:   false,
		},
		{
			name: "has API requests",
			setup: func(u *SubscriptionUsage) {
				u.IncrementAPIRequests(1)
			},
			want: true,
		},
		{
			name: "has storage",
			setup: func(u *SubscriptionUsage) {
				u.IncrementStorageUsed(1000)
			},
			want: true,
		},
		{
			name: "has users",
			setup: func(u *SubscriptionUsage) {
				u.IncrementUsersCount()
			},
			want: true,
		},
		{
			name: "has projects",
			setup: func(u *SubscriptionUsage) {
				u.IncrementProjectsCount()
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage, _ := NewSubscriptionUsage(1, now)
			tt.setup(usage)

			if got := usage.HasUsage(); got != tt.want {
				t.Errorf("HasUsage() = %v, want %v", got, tt.want)
			}
		})
	}
}
