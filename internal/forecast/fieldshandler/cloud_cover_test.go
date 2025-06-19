package fieldshandler

import (
	"math"
	"testing"
)

func TestCloudCover(t *testing.T) {
	testCases := []struct {
		name     string
		lcc      float64
		mcc      float64
		hcc      float64
		expected float64
	}{
		{
			name:     "Normal case",
			lcc:      0.5, // 50%
			mcc:      0.2, // 20%
			hcc:      0.1, // 10%
			expected: 64.0,
		},
		{
			name:     "Zero cloud cover",
			lcc:      0.0,
			mcc:      0.0,
			hcc:      0.0,
			expected: 0.0,
		},
		{
			name:     "Full cloud cover from lcc",
			lcc:      1.0,
			mcc:      0.5,
			hcc:      0.5,
			expected: 100.0,
		},
		{
			name:     "Full cloud cover from all layers",
			lcc:      1.0,
			mcc:      1.0,
			hcc:      1.0,
			expected: 100.0,
		},
		{
			name:     "Only medium clouds",
			lcc:      0.0,
			mcc:      0.5,
			hcc:      0.0,
			expected: 50.0,
		},
		{
			name:     "Only high clouds",
			lcc:      0.0,
			mcc:      0.0,
			hcc:      0.8,
			expected: 80.0,
		},
		{
			name:     "Medium and high clouds",
			lcc:      0.0,
			mcc:      0.5,
			hcc:      0.5,
			expected: 75.0, // 0 + 0.5*(1-0) + 0.5*(1-0)*(1-0.5) = 0.5 + 0.25 = 0.75
		},
		{
			name:     "Invalid inputs (greater than 1)",
			lcc:      1.1,
			mcc:      1.2,
			hcc:      1.3,
			expected: 100.0,
		},
		{
			name:     "Invalid inputs (less than 0)",
			lcc:      -0.1,
			mcc:      -0.2,
			hcc:      -0.3,
			expected: 0.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := cloudCover(tc.lcc, tc.mcc, tc.hcc)
			if math.Abs(result-tc.expected) > 0.001 { // Using a tolerance for float comparison
				t.Errorf("calculateTotalCloudCover(%f, %f, %f) = %f; want %f", tc.lcc, tc.mcc, tc.hcc, result, tc.expected)
			}
		})
	}
} 