package codebook

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeoutsReloadDelay(t *testing.T) {
	referenceReloadInterval := 10 * time.Second
	expetedResult := map[time.Duration]bool{
		0 * time.Second:  true,
		5 * time.Second:  true,
		8 * time.Second:  true,
		10 * time.Second: true,
		15 * time.Second: false,
	}

	for reloadDelay, expected := range expetedResult {
		timeouts := Timeouts{
			ReloadInterval: referenceReloadInterval,
			ReloadDelay:    reloadDelay,
		}
		assert.Equal(t, expected, timeouts.check() == nil)
	}
}

func TestTimeoutsRanomizerValue(t *testing.T) {
	expetedResult := map[float64]bool{
		0.5:  true,
		0.0:  true,
		-0.5: false,
		1.1:  false,
	}

	for value, expected := range expetedResult {
		timeouts := Timeouts{
			Ranomizer: value,
		}
		assert.Equal(t, expected, timeouts.check() == nil)
	}
}
