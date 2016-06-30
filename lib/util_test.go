package health

import (
	"testing"
	"time"

	"github.com/blendlabs/go-assert"
)

var (
	testCases = map[string]time.Duration{
		"1h":           1 * time.Hour,
		"1h1m":         (1 * time.Hour) + (1 * time.Minute),
		"1h1m1s":       (1 * time.Hour) + (1 * time.Minute) + (1 * time.Second),
		"1s100ms":      (1 * time.Second) + (100 * time.Millisecond),
		"1s100ms50µs":  (1 * time.Second) + (100 * time.Millisecond) + (50 * time.Microsecond),
		"7h6m5s4ms3µs": (7 * time.Hour) + (6 * time.Minute) + (5 * time.Second) + (4 * time.Millisecond) + (3 * time.Microsecond),
		"5m":           (5 * time.Minute),
		"5s":           (5 * time.Second),
		"5ms":          (5 * time.Millisecond),
		"5µs":          (5 * time.Microsecond),
	}
)

func TestSumDurations(t *testing.T) {
	assert := assert.New(t)

	assert.Equal(time.Hour+time.Minute+time.Second, SumDurations(time.Hour, time.Minute, time.Second))
}

func TestParseDuration(t *testing.T) {
	assert := assert.New(t)

	for label, value := range testCases {
		assert.Equal(value, ParseDuration(label))
	}
}

func TestParseDurationRoundTrip(t *testing.T) {
	assert := assert.New(t)

	for _, value := range testCases {
		assert.Equal(value, ParseDuration(FormatDuration(value)))
	}

	for label := range testCases {
		assert.Equal(label, FormatDuration(ParseDuration(label)))
	}
}

func TestRoundDuration(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(time.Millisecond, RoundDuration(time.Millisecond+time.Microsecond, time.Millisecond))
}
