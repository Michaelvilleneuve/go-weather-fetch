package fieldshandler

import (
	"math"
	"testing"
)

func TestComfortIndex(t *testing.T) {
	testCases := []struct {
		name     string
		t2m      float64
		u10      float64
		v10      float64
		r2       float64
		expected float64
	}{
		{
			name:     "Normal conditions",
			t2m:      293.15, // 20°C
			u10:      2,
			v10:      2,
			r2:       50,
			expected: 5.87,
		},
		{
			name:     "Cold conditions",
			t2m:      273.15, // 0°C
			u10:      5,
			v10:      0,
			r2:       60,
			expected: 2.76,
		},
		{
			name:     "Hot and humid conditions",
			t2m:      308.15, // 35°C
			u10:      1,
			v10:      1,
			r2:       80,
			expected: 9.33,
		},
		{
			name:     "Extreme cold",
			t2m:      253.15, // -20°C
			u10:      10,
			v10:      10,
			r2:       50,
			expected: 1.0,
		},
		{
			name:     "Extreme hot",
			t2m:      313.15, // 40°C
			u10:      0,
			v10:      0,
			r2:       90,
			expected: 10.0,
		},
		{
			name:     "Invalid t2m low",
			t2m:      199,
			u10:      0,
			v10:      0,
			r2:       50,
			expected: 5.0,
		},
		{
			name:     "Invalid t2m high",
			t2m:      351,
			u10:      0,
			v10:      0,
			r2:       50,
			expected: 5.0,
		},
		{
			name:     "Invalid r2 low",
			t2m:      293.15,
			u10:      0,
			v10:      0,
			r2:       -1,
			expected: 5.0,
		},
		{
			name:     "Invalid r2 high",
			t2m:      293.15,
			u10:      0,
			v10:      0,
			r2:       101,
			expected: 5.0,
		},
		{
			name:     "Invalid u10",
			t2m:      293.15,
			u10:      101,
			v10:      0,
			r2:       50,
			expected: 5.0,
		},
		{
			name:     "Invalid v10",
			t2m:      293.15,
			u10:      0,
			v10:      -101,
			r2:       50,
			expected: 5.0,
		},
		{
			name:     "Zero wind",
			t2m:      293.15, // 20°C
			u10:      0,
			v10:      0,
			r2:       50,
			expected: 6.12,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := comfortIndex(tc.t2m, tc.u10, tc.v10, tc.r2)
			if math.Abs(result-tc.expected) > 0.01 { // Using a tolerance for float comparison
				t.Errorf("comfortIndex(%f, %f, %f, %f) = %f; want %f", tc.t2m, tc.u10, tc.v10, tc.r2, result, tc.expected)
			}
		})
	}
} 