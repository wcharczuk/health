package util

import "strconv"

// Parse contains utilities to parse strings.
var Parse = new(parseUtil)

type parseUtil struct{}

// ParseFloat64 parses a float64
func (pu parseUtil) Float64(input string) float64 {
	result, err := strconv.ParseFloat(input, 64)
	if err != nil {
		return 0.0
	}
	return result
}

// ParseFloat32 parses a float32
func (pu parseUtil) Float32(input string) float32 {
	result, err := strconv.ParseFloat(input, 32)
	if err != nil {
		return 0.0
	}
	return float32(result)
}

// ParseInt parses an int
func (pu parseUtil) Int(input string) int {
	result, err := strconv.Atoi(input)
	if err != nil {
		return 0
	}
	return result
}

// ParseInt32 parses an int
func (pu parseUtil) Int32(input string) int32 {
	result, err := strconv.Atoi(input)
	if err != nil {
		return 0
	}
	return int32(result)
}

// ParseInt64 parses an int64
func (pu parseUtil) Int64(input string) int64 {
	result, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		return int64(0)
	}
	return result
}
