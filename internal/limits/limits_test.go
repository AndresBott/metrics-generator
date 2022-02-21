package limits

import (
	"testing"
	"time"
)

func TestSetSleepDuration(t *testing.T) {

	t.Run("2 req/h expect 30min wait", func(t *testing.T) {
		cfg := Config{}

		cfg.SetRequestsHour(2)

		want := 30 * time.Minute
		got := cfg.sleepDuration

		if got != want {
			t.Errorf("unexpeced duration, got %v, want %v ", got, want)
		}
	})

	t.Run("3600req/h expect 1sec wait", func(t *testing.T) {
		cfg := Config{}
		cfg.SetRequestsHour(3600)

		want := 1 * time.Second
		got := cfg.sleepDuration

		if got != want {
			t.Errorf("unexpeced duration, got %v, want %v ", got, want)
		}
	})

}
