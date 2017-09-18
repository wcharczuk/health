package logger

import "time"

const (
	// NanosecondsPerSecond is the number of nanoseconds in a second.
	NanosecondsPerSecond = time.Second / time.Nanosecond
)

// Seconds returns a duration as seconds.
func Seconds(d time.Duration) float64 {
	return float64(d) / float64(time.Second)
}

// Milliseconds returns a duration as milliseconds.
func Milliseconds(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

// Microseconds returns a duration as microseconds.
func Microseconds(d time.Duration) float64 {
	return float64(d) / float64(time.Microsecond)
}

// UnixNano returns both the unix timestamp (in seconds), and the
// nanosecond remainder.
func UnixNano(t time.Time) (int64, int64) {
	unix := t.Unix() //seconds
	unixSecondsAsNanoseconds := int64(time.Duration(unix) * NanosecondsPerSecond)
	nano := t.UnixNano() - unixSecondsAsNanoseconds
	return unix, nano
}

// SumOfDuration adds all the values of a slice together
func SumOfDuration(values []time.Duration) time.Duration {
	total := time.Duration(0)
	for x := 0; x < len(values); x++ {
		total += values[x]
	}

	return total
}

// MeanOfDuration gets the average of a slice of numbers
func MeanOfDuration(input []time.Duration) time.Duration {
	if len(input) == 0 {
		return 0
	}

	sum := SumOfDuration(input)
	mean := uint64(sum) / uint64(len(input))
	return time.Duration(mean)
}
