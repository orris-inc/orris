package mapper

import (
	"errors"
	"fmt"
	"testing"
)

// Test structs
type testInput struct {
	Value int
}

type testOutput struct {
	Result string
}

// =============================================================================
// MapSliceWithError Tests
// =============================================================================

func TestMapSliceWithError(t *testing.T) {
	tests := []struct {
		name        string
		input       []int
		mapFunc     func(int) (string, error)
		want        []string
		wantErr     bool
		errContains string
	}{
		{
			name:    "nil input returns nil",
			input:   nil,
			mapFunc: func(i int) (string, error) { return fmt.Sprintf("%d", i), nil },
			want:    nil,
			wantErr: false,
		},
		{
			name:    "empty slice returns empty slice",
			input:   []int{},
			mapFunc: func(i int) (string, error) { return fmt.Sprintf("%d", i), nil },
			want:    []string{},
			wantErr: false,
		},
		{
			name:    "successful mapping",
			input:   []int{1, 2, 3},
			mapFunc: func(i int) (string, error) { return fmt.Sprintf("num_%d", i), nil },
			want:    []string{"num_1", "num_2", "num_3"},
			wantErr: false,
		},
		{
			name:  "first element returns error",
			input: []int{1, 2, 3},
			mapFunc: func(i int) (string, error) {
				return "", errors.New("mapping failed")
			},
			want:        nil,
			wantErr:     true,
			errContains: "mapping failed",
		},
		{
			name:  "middle element returns error",
			input: []int{1, 2, 3, 4, 5},
			mapFunc: func(i int) (string, error) {
				if i == 3 {
					return "", errors.New("error at element 3")
				}
				return fmt.Sprintf("num_%d", i), nil
			},
			want:        nil,
			wantErr:     true,
			errContains: "error at element 3",
		},
		{
			name:  "last element returns error",
			input: []int{1, 2, 3},
			mapFunc: func(i int) (string, error) {
				if i == 3 {
					return "", errors.New("error at last element")
				}
				return fmt.Sprintf("num_%d", i), nil
			},
			want:        nil,
			wantErr:     true,
			errContains: "error at last element",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapSliceWithError(tt.input, tt.mapFunc)

			if tt.wantErr {
				if err == nil {
					t.Errorf("MapSliceWithError() expected error, got nil")
					return
				}
				if tt.errContains != "" && err.Error() != tt.errContains {
					t.Errorf("MapSliceWithError() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("MapSliceWithError() unexpected error: %v", err)
				return
			}

			if tt.input == nil {
				if got != nil {
					t.Errorf("MapSliceWithError() = %v, want nil", got)
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("MapSliceWithError() length = %d, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("MapSliceWithError()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestMapSliceWithError_WithStructs(t *testing.T) {
	tests := []struct {
		name    string
		input   []testInput
		mapFunc func(testInput) (testOutput, error)
		want    []testOutput
		wantErr bool
	}{
		{
			name:  "struct mapping successful",
			input: []testInput{{Value: 1}, {Value: 2}, {Value: 3}},
			mapFunc: func(in testInput) (testOutput, error) {
				return testOutput{Result: fmt.Sprintf("result_%d", in.Value)}, nil
			},
			want:    []testOutput{{Result: "result_1"}, {Result: "result_2"}, {Result: "result_3"}},
			wantErr: false,
		},
		{
			name:  "struct mapping with error",
			input: []testInput{{Value: 1}, {Value: -1}, {Value: 3}},
			mapFunc: func(in testInput) (testOutput, error) {
				if in.Value < 0 {
					return testOutput{}, errors.New("negative value not allowed")
				}
				return testOutput{Result: fmt.Sprintf("result_%d", in.Value)}, nil
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapSliceWithError(tt.input, tt.mapFunc)

			if tt.wantErr {
				if err == nil {
					t.Errorf("MapSliceWithError() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("MapSliceWithError() unexpected error: %v", err)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("MapSliceWithError() length = %d, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i].Result != tt.want[i].Result {
					t.Errorf("MapSliceWithError()[%d].Result = %v, want %v", i, got[i].Result, tt.want[i].Result)
				}
			}
		})
	}
}

// =============================================================================
// MapSlicePtrSkipNil Tests
// =============================================================================

func TestMapSlicePtrSkipNil(t *testing.T) {
	// Helper function to create int pointer
	intPtr := func(i int) *int { return &i }
	// Helper function to create string pointer
	strPtr := func(s string) *string { return &s }

	t.Run("nil input returns nil", func(t *testing.T) {
		var input []*int = nil
		mapFunc := func(i *int) *string {
			s := fmt.Sprintf("%d", *i)
			return &s
		}

		got := MapSlicePtrSkipNil(input, mapFunc)
		if got != nil {
			t.Errorf("MapSlicePtrSkipNil() = %v, want nil", got)
		}
	})

	t.Run("empty slice returns empty slice", func(t *testing.T) {
		input := []*int{}
		mapFunc := func(i *int) *string {
			s := fmt.Sprintf("%d", *i)
			return &s
		}

		got := MapSlicePtrSkipNil(input, mapFunc)
		if got == nil {
			t.Errorf("MapSlicePtrSkipNil() = nil, want empty slice")
			return
		}
		if len(got) != 0 {
			t.Errorf("MapSlicePtrSkipNil() length = %d, want 0", len(got))
		}
	})

	t.Run("normal mapping without nil", func(t *testing.T) {
		input := []*int{intPtr(1), intPtr(2), intPtr(3)}
		mapFunc := func(i *int) *string {
			s := fmt.Sprintf("num_%d", *i)
			return &s
		}

		got := MapSlicePtrSkipNil(input, mapFunc)
		want := []*string{strPtr("num_1"), strPtr("num_2"), strPtr("num_3")}

		if len(got) != len(want) {
			t.Errorf("MapSlicePtrSkipNil() length = %d, want %d", len(got), len(want))
			return
		}

		for i := range got {
			if *got[i] != *want[i] {
				t.Errorf("MapSlicePtrSkipNil()[%d] = %v, want %v", i, *got[i], *want[i])
			}
		}
	})

	t.Run("input contains nil elements - skipped", func(t *testing.T) {
		input := []*int{intPtr(1), nil, intPtr(3), nil, intPtr(5)}
		mapFunc := func(i *int) *string {
			s := fmt.Sprintf("num_%d", *i)
			return &s
		}

		got := MapSlicePtrSkipNil(input, mapFunc)
		want := []*string{strPtr("num_1"), strPtr("num_3"), strPtr("num_5")}

		if len(got) != len(want) {
			t.Errorf("MapSlicePtrSkipNil() length = %d, want %d", len(got), len(want))
			return
		}

		for i := range got {
			if *got[i] != *want[i] {
				t.Errorf("MapSlicePtrSkipNil()[%d] = %v, want %v", i, *got[i], *want[i])
			}
		}
	})

	t.Run("map function returns nil - skipped", func(t *testing.T) {
		input := []*int{intPtr(1), intPtr(2), intPtr(3)}
		mapFunc := func(i *int) *string {
			if *i == 2 {
				return nil // Return nil for value 2
			}
			s := fmt.Sprintf("num_%d", *i)
			return &s
		}

		got := MapSlicePtrSkipNil(input, mapFunc)
		want := []*string{strPtr("num_1"), strPtr("num_3")}

		if len(got) != len(want) {
			t.Errorf("MapSlicePtrSkipNil() length = %d, want %d", len(got), len(want))
			return
		}

		for i := range got {
			if *got[i] != *want[i] {
				t.Errorf("MapSlicePtrSkipNil()[%d] = %v, want %v", i, *got[i], *want[i])
			}
		}
	})

	t.Run("mixed case - nil input and nil output", func(t *testing.T) {
		input := []*int{intPtr(1), nil, intPtr(2), intPtr(3), nil, intPtr(4)}
		mapFunc := func(i *int) *string {
			// Return nil for even numbers
			if *i%2 == 0 {
				return nil
			}
			s := fmt.Sprintf("odd_%d", *i)
			return &s
		}

		got := MapSlicePtrSkipNil(input, mapFunc)
		// Expected: 1 -> "odd_1", nil -> skip, 2 -> nil -> skip, 3 -> "odd_3", nil -> skip, 4 -> nil -> skip
		want := []*string{strPtr("odd_1"), strPtr("odd_3")}

		if len(got) != len(want) {
			t.Errorf("MapSlicePtrSkipNil() length = %d, want %d", len(got), len(want))
			return
		}

		for i := range got {
			if *got[i] != *want[i] {
				t.Errorf("MapSlicePtrSkipNil()[%d] = %v, want %v", i, *got[i], *want[i])
			}
		}
	})

	t.Run("all inputs are nil", func(t *testing.T) {
		input := []*int{nil, nil, nil}
		mapFunc := func(i *int) *string {
			s := fmt.Sprintf("num_%d", *i)
			return &s
		}

		got := MapSlicePtrSkipNil(input, mapFunc)
		if len(got) != 0 {
			t.Errorf("MapSlicePtrSkipNil() length = %d, want 0", len(got))
		}
	})

	t.Run("all outputs are nil", func(t *testing.T) {
		input := []*int{intPtr(1), intPtr(2), intPtr(3)}
		mapFunc := func(i *int) *string {
			return nil // Always return nil
		}

		got := MapSlicePtrSkipNil(input, mapFunc)
		if len(got) != 0 {
			t.Errorf("MapSlicePtrSkipNil() length = %d, want 0", len(got))
		}
	})
}

func TestMapSlicePtrSkipNil_WithStructs(t *testing.T) {
	t.Run("struct pointer mapping", func(t *testing.T) {
		input := []*testInput{
			{Value: 1},
			nil,
			{Value: 2},
			{Value: -1}, // Will be filtered by map func
			{Value: 3},
		}

		mapFunc := func(in *testInput) *testOutput {
			if in.Value < 0 {
				return nil // Filter out negative values
			}
			return &testOutput{Result: fmt.Sprintf("result_%d", in.Value)}
		}

		got := MapSlicePtrSkipNil(input, mapFunc)

		// Expected: 1 -> result_1, nil -> skip, 2 -> result_2, -1 -> nil -> skip, 3 -> result_3
		want := []*testOutput{
			{Result: "result_1"},
			{Result: "result_2"},
			{Result: "result_3"},
		}

		if len(got) != len(want) {
			t.Errorf("MapSlicePtrSkipNil() length = %d, want %d", len(got), len(want))
			return
		}

		for i := range got {
			if got[i].Result != want[i].Result {
				t.Errorf("MapSlicePtrSkipNil()[%d].Result = %v, want %v", i, got[i].Result, want[i].Result)
			}
		}
	})
}
