package metrics

import (
	"math/rand"
	"testing"
)

// this test does not call the actual implementation because to instantiate we would need
// to create a struct with an unexported value from another package
// additionally due to the random nature of the function, the outcome is not predictable
// this test is used only to visually verify the behaviour of the implementation with different
// input values
func TestShouldFailRequest(t *testing.T) {

	t.Skip("test not reliable, only used to verify numbers on the implementation")

	tcs := []struct {
		name  string
		value float64
	}{
		{
			name:  "100% percent",
			value: 100,
		},
		{
			name:  "50% percent",
			value: 50,
		},
		{
			name:  "0.1% percent",
			value: 0.1,
		},
		{
			name:  "0.05% percent",
			value: 0.05,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {

			total := 100000
			errors := 0
			f := func() bool {
				f := float64(rand.Intn(100000)) / 1000
				return f < tc.value
			}
			for i := 0; i < total; i++ {
				v := f()
				if v {
					errors++
				}
			}

			got := (float64(errors) / float64(total)) * 100
			t.Logf("input:%f got:%f", tc.value, got)

		})
	}
}
