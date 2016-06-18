package health

import (
	"fmt"
	"math"
	"time"
)

// AverageDuration returns the average of a list of durations.
func AverageDuration(durations ...time.Duration) time.Duration {
	var accum time.Duration

	for _, d := range durations {
		accum += d
	}

	return accum / time.Duration(len(durations))
}

// MinDuration returns the smallest duration.
func MinDuration(durations ...time.Duration) time.Duration {
	var min time.Duration
	for _, d := range durations {
		if min == 0 || d < min {
			min = d
		}
	}
	return min
}

// MaxDuration returns the maximum duration.
func MaxDuration(durations ...time.Duration) time.Duration {
	var max time.Duration
	for _, d := range durations {
		if d > max {
			max = d
		}
	}
	return max
}

// FormatDuration prints a duration as a string.
func FormatDuration(d time.Duration) string {
	if d > time.Hour {
		hours := d / time.Hour
		hoursRemainder := d - (hours * time.Hour)
		minutes := hoursRemainder / time.Minute
		minuteRemainder := hoursRemainder - (minutes * time.Minute)
		seconds := minuteRemainder / time.Second
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	} else if d > time.Minute {
		minutes := d / time.Minute
		minuteRemainder := d - (minutes * time.Minute)
		seconds := minuteRemainder / time.Second
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	} else if d > time.Second {
		seconds := d / time.Second
		secondsRemainder := d - (seconds * time.Second)
		milliseconds := secondsRemainder / time.Millisecond
		return fmt.Sprintf("%d.%ds", seconds, milliseconds)
	} else if d > time.Millisecond {
		milliseconds := d / time.Millisecond
		return fmt.Sprintf("%dms", milliseconds)
	} else {
		microseconds := d / time.Microsecond
		return fmt.Sprintf("%dµs", microseconds)
	}
}

// Round rounds  a float to a given places.
func Round(input float64, places int) float64 {
	sign := 1.0
	if input < 0 {
		sign = -1
		input *= -1
	}

	precision := math.Pow(10, float64(places))
	digit := input * precision
	_, decimal := math.Modf(digit)

	var rounded float64
	if decimal >= 0.5 {
		rounded = math.Ceil(digit)
	} else {
		rounded = math.Floor(digit)
	}

	return rounded / precision * sign
}

// RoundToInt rounds an float to an integer.
// Round conditionally rounds up or down depending on how close
// The 10ths and 100ths ... values are to 0,1.
func RoundToInt(input float64) int {
	return int(Round(input, 0))
}

// HostsFlag is a variable length flag.
type HostsFlag []string

// String returns the help text for the flag.
func (h *HostsFlag) String() string {
	return "Hosts to ping."
}

// Set adds a value to the flag set.
func (h *HostsFlag) Set(value string) error {
	*h = append(*h, value)
	return nil
}

// Duratoins is a helper list to sort time.Durations
type durations []time.Duration

func (dl durations) Len() int {
	return len(dl)
}

func (dl durations) Less(i, j int) bool {
	return dl[i] < dl[j]
}

func (dl durations) Swap(i, j int) {
	dl[i], dl[j] = dl[j], dl[i]
}
