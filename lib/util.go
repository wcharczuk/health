package health

import (
	"fmt"
	"math"
	"strconv"
	"time"

	util "github.com/blendlabs/go-util"
)

const (
	// Spark0 is a spark-line token.
	Spark0 = "▁"
	// Spark1 is a spark-line token.
	Spark1 = "▂"
	// Spark2 is a spark-line token.
	Spark2 = "▃"
	// Spark3 is a spark-line token.
	Spark3 = "▅"
	// Spark4 is a spark-line token.
	Spark4 = "▇"
)

// FormatSparklines returns sparklines for the given value.
func FormatSparklines(values []float64, optionalMax ...float64) string {
	var max float64
	if len(optionalMax) > 0 {
		max = optionalMax[0]
	} else {
		max = util.Math.Max(values)
	}
	var normalized []float64
	for _, v := range values {
		normalized = append(normalized, v/max)
	}
	var output string
	for _, nv := range normalized {
		if nv > 0.8 {
			output = output + Spark4
		} else if nv > 0.6 {
			output = output + Spark3
		} else if nv > 0.4 {
			output = output + Spark2
		} else if nv > 0.2 {
			output = output + Spark1
		} else {
			output = output + Spark0
		}
	}
	return output
}

// Duration implements a custom json marshaller.
type Duration time.Duration

// AsTimeDuration returns the duration as a time.Duration.
func (d Duration) AsTimeDuration() time.Duration {
	return time.Duration(d)
}

// MarshalJSON marshals the Duration using `FormatDuration`.
func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(FormatDuration(d.AsTimeDuration())), nil
}

// UnmarshalJSON parses a duration string from a json blob.
func (d *Duration) UnmarshalJSON(data []byte) error {
	return nil
}

// RoundTo calls `RoundDuration` for the Duration.
func (d Duration) RoundTo(roundTo time.Duration) time.Duration {
	return RoundDuration(time.Duration(d), roundTo)
}

// SumDurations returns the sum of a given variadic set of durations.
func SumDurations(durations ...time.Duration) time.Duration {
	var accum time.Duration
	for _, d := range durations {
		accum += d
	}
	return accum
}

// AverageDuration returns the average of a list of durations.
func AverageDuration(durations ...time.Duration) time.Duration {
	total := SumDurations(durations...)
	return total / time.Duration(len(durations))
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

// ExplodeDuration returns all the constitent parts of a time.Duration.
func ExplodeDuration(duration time.Duration) (
	hours time.Duration,
	minutes time.Duration,
	seconds time.Duration,
	milliseconds time.Duration,
	microseconds time.Duration,
) {
	hours = duration / time.Hour
	hoursRemainder := duration - (hours * time.Hour)
	minutes = hoursRemainder / time.Minute
	minuteRemainder := hoursRemainder - (minutes * time.Minute)
	seconds = minuteRemainder / time.Second
	secondsRemainder := minuteRemainder - (seconds * time.Second)
	milliseconds = secondsRemainder / time.Millisecond
	millisecondsRemainder := secondsRemainder - (milliseconds * time.Millisecond)
	microseconds = millisecondsRemainder / time.Microsecond
	return
}

// FormatDuration prints a duration as a string.
func FormatDuration(duration time.Duration) string {
	hours, minutes, seconds, milliseconds, microseconds := ExplodeDuration(duration)
	var value string
	if hours > 0 {
		value = fmt.Sprintf("%dh", hours)
	}
	if minutes > 0 {
		value = value + fmt.Sprintf("%dm", minutes)
	}
	if seconds > 0 {
		value = value + fmt.Sprintf("%ds", seconds)
	}
	if milliseconds > 0 {
		value = value + fmt.Sprintf("%dms", milliseconds)
	}
	if microseconds > 0 {
		value = value + fmt.Sprintf("%dµs", microseconds)
	}
	return value
}

// RoundDuration rounds a duration to the given place.
func RoundDuration(duration, roundTo time.Duration) time.Duration {
	hours, minutes, seconds, milliseconds, microseconds := ExplodeDuration(duration)
	hours = hours * time.Hour
	minutes = minutes * time.Minute
	seconds = seconds * time.Second
	milliseconds = milliseconds * time.Millisecond
	microseconds = microseconds * time.Microsecond

	var total time.Duration
	if hours >= roundTo {
		total = total + hours
	}
	if minutes >= roundTo {
		total = total + minutes
	}
	if seconds >= roundTo {
		total = total + seconds
	}
	if milliseconds >= roundTo {
		total = total + milliseconds
	}
	if microseconds >= roundTo {
		total = total + microseconds
	}

	return total
}

// ParseDuration reverses `FormatDuration`.
func ParseDuration(duration string) time.Duration {
	integerValue, err := strconv.ParseInt(duration, 10, 64)
	if err == nil {
		println("passes parseint")
		return time.Duration(integerValue)
	}

	var hours int64
	var minutes int64
	var seconds int64
	var milliseconds int64
	var microseconds int64

	state := 0
	lastIndex := len([]rune(duration)) - 1

	var numberValue string
	var labelValue string

	var consumeValues = func() {
		switch labelValue {
		case "h":
			hours = ParseInt64(numberValue)
		case "m":
			minutes = ParseInt64(numberValue)
		case "s":
			seconds = ParseInt64(numberValue)
		case "ms":
			milliseconds = ParseInt64(numberValue)
		case "µs":
			microseconds = ParseInt64(numberValue)
		}
	}

	for index, c := range []rune(duration) {
		switch state {
		case 0:
			if IsNumber(c) {
				numberValue = numberValue + string(c)
			} else {
				labelValue = string(c)
				if index == lastIndex {
					consumeValues()
				} else {
					state = 1
				}
			}
		case 1:
			if IsNumber(c) {
				consumeValues()
				numberValue = string(c)
				state = 0
			} else if index == lastIndex {
				labelValue = labelValue + string(c)
				consumeValues()
			} else {
				labelValue = labelValue + string(c)
			}
		}
	}

	return (time.Duration(hours) * time.Hour) +
		(time.Duration(minutes) * time.Minute) +
		(time.Duration(seconds) * time.Second) +
		(time.Duration(milliseconds) * time.Millisecond) +
		(time.Duration(microseconds) * time.Microsecond)
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

// IsNumber returns if a rune is in the number range.
func IsNumber(c rune) bool {
	return c >= rune('0') && c <= rune('9')
}

// ParseInt64 parses an int64
func ParseInt64(input string) int64 {
	result, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		return int64(0)
	}
	return result
}
