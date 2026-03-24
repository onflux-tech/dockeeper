package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBackoffBase(t *testing.T) {
	d := backoff(1)
	assert.GreaterOrEqual(t, d, 750*time.Millisecond, "first attempt should be ~1s ±25%%")
	assert.LessOrEqual(t, d, 1250*time.Millisecond)
}

func TestBackoffGrows(t *testing.T) {
	d1 := backoff(1)
	d5 := backoff(5)
	assert.Greater(t, d5, d1, "backoff should grow with attempts")
}

func TestBackoffMaxCap(t *testing.T) {
	d := backoff(100)
	// With 25% jitter, max should be at most backoffMax * 1.25
	ceiling := time.Duration(float64(backoffMax) * 1.25)
	assert.LessOrEqual(t, d, ceiling, "backoff should be capped")
}

func TestErrServerGone(t *testing.T) {
	assert.NotNil(t, ErrServerGone)
	assert.Equal(t, "server record deleted on manager", ErrServerGone.Error())
}
