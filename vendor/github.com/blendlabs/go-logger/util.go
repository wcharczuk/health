package logger

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// GetIP gets the origin/client ip for a request.
// X-FORWARDED-FOR is checked. If multiple IPs are included the first one is returned
// X-REAL-IP is checked. If multiple IPs are included the first one is returned
// Finally r.RemoteAddr is used
// Only benevolent services will allow access to the real IP.
func GetIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	tryHeader := func(key string) (string, bool) {
		if headerVal := r.Header.Get(key); len(headerVal) > 0 {
			if !strings.ContainsRune(headerVal, ',') {
				return headerVal, true
			}
			return strings.SplitN(headerVal, ",", 2)[0], true
		}
		return "", false
	}

	for _, header := range []string{"X-FORWARDED-FOR", "X-REAL-IP"} {
		if headerVal, ok := tryHeader(header); ok {
			return headerVal
		}
	}

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

// ErrTypeConversion represents an error in marshalling a type, typically event state.
var ErrTypeConversion = errors.New("Invalid event state type conversion")

func stateAsRequest(state interface{}) (*http.Request, error) {
	if typed, isTyped := state.(*http.Request); isTyped {
		return typed, nil
	}
	return nil, ErrTypeConversion
}

func stateAsInteger(state interface{}) (int, error) {
	if typed, isTyped := state.(int); isTyped {
		return typed, nil
	}
	return 0, ErrTypeConversion
}

func stateAsAnsiColorCode(state interface{}) (AnsiColorCode, error) {
	if typed, isTyped := state.(AnsiColorCode); isTyped {
		return typed, nil
	}
	return ColorReset, ErrTypeConversion
}

func stateAsEvent(state interface{}) (Event, error) {
	if typed, isTyped := state.(Event); isTyped {
		return typed, nil
	}
	return EventNone, ErrTypeConversion
}

func stateAsTime(state interface{}) (time.Time, error) {
	if typed, isTyped := state.(time.Time); isTyped {
		return typed, nil
	}
	return time.Time{}, ErrTypeConversion
}

func stateAsTimeSource(state interface{}) (TimeSource, error) {
	if typed, isTyped := state.(TimeSource); isTyped {
		return typed, nil
	}
	return SystemClock, ErrTypeConversion
}

func stateAsDuration(state interface{}) (time.Duration, error) {
	if typed, isTyped := state.(time.Duration); isTyped {
		return typed, nil
	}
	return 0, ErrTypeConversion
}

func stateAsString(state interface{}) (string, error) {
	if typed, isTyped := state.(string); isTyped {
		return typed, nil
	}
	return "", ErrTypeConversion
}

func stateAsBytes(state interface{}) ([]byte, error) {
	if typed, isTyped := state.([]byte); isTyped {
		return typed, nil
	}
	return nil, ErrTypeConversion
}

func envFlagIsSet(flagName string, defaultValue bool) bool {
	flagValue := os.Getenv(flagName)
	if len(flagValue) > 0 {
		if strings.ToUpper(flagValue) == "TRUE" || flagValue == "1" {
			return true
		}
		return false
	}
	return defaultValue
}

func envFlagInt(flagName string, defaultValue int) int {
	flagValue := os.Getenv(flagName)
	if len(flagValue) > 0 {
		value, err := strconv.Atoi(flagValue)
		if err != nil {
			return defaultValue
		}
		return value
	}
	return defaultValue
}

func envFlagInt64(flagName string, defaultValue int64) int64 {
	flagValue := os.Getenv(flagName)
	if len(flagValue) > 0 {
		value, err := strconv.ParseInt(flagValue, 10, 64)
		if err != nil {
			return defaultValue
		}
		return value
	}
	return defaultValue
}

var (
	// LowerA is the ascii int value for 'a'
	LowerA = uint('a')
	// LowerZ is the ascii int value for 'z'
	LowerZ = uint('z')

	lowerDiff = (LowerZ - LowerA)
)

// HasPrefixCaseInsensitive returns if a corpus has a prefix regardless of casing.
func HasPrefixCaseInsensitive(corpus, prefix string) bool {
	corpusLen := len(corpus)
	prefixLen := len(prefix)

	if corpusLen < prefixLen {
		return false
	}

	for x := 0; x < prefixLen; x++ {
		charCorpus := uint(corpus[x])
		charPrefix := uint(prefix[x])

		if charCorpus-LowerA <= lowerDiff {
			charCorpus = charCorpus - 0x20
		}

		if charPrefix-LowerA <= lowerDiff {
			charPrefix = charPrefix - 0x20
		}
		if charCorpus != charPrefix {
			return false
		}
	}
	return true
}

// HasSuffixCaseInsensitive returns if a corpus has a suffix regardless of casing.
func HasSuffixCaseInsensitive(corpus, suffix string) bool {
	corpusLen := len(corpus)
	suffixLen := len(suffix)

	if corpusLen < suffixLen {
		return false
	}

	for x := 0; x < suffixLen; x++ {
		charCorpus := uint(corpus[corpusLen-(x+1)])
		charSuffix := uint(suffix[suffixLen-(x+1)])

		if charCorpus-LowerA <= lowerDiff {
			charCorpus = charCorpus - 0x20
		}

		if charSuffix-LowerA <= lowerDiff {
			charSuffix = charSuffix - 0x20
		}
		if charCorpus != charSuffix {
			return false
		}
	}
	return true
}

// CaseInsensitiveEquals compares two strings regardless of case.
func CaseInsensitiveEquals(a, b string) bool {
	aLen := len(a)
	bLen := len(b)
	if aLen != bLen {
		return false
	}

	for x := 0; x < aLen; x++ {
		charA := uint(a[x])
		charB := uint(b[x])

		if charA-LowerA <= lowerDiff {
			charA = charA - 0x20
		}
		if charB-LowerA <= lowerDiff {
			charB = charB - 0x20
		}
		if charA != charB {
			return false
		}
	}

	return true
}

// UUIDv4 returns a v4 uuid short string.
func UUIDv4() string {
	uuid := make([]byte, 16)
	rand.Read(uuid)
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // set version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // set variant 10
	return fmt.Sprintf("%x", uuid[:])
}
