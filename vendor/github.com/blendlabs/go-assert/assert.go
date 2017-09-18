package assert

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	// RED is the ansi escape code fragment for red.
	RED = "31"
	// BLUE is the ansi escape code fragment for blue.
	BLUE = "94"
	// GREEN is the ansi escape code fragment for green.
	GREEN = "32"
	// YELLOW is the ansi escape code fragment for yellow.
	YELLOW = "33"
	// WHITE is the ansi escape code fragment for white.
	WHITE = "37"
	// GRAY is the ansi escape code fragment for gray.
	GRAY = "90"

	// EMPTY is a constant for the empty (0 length) string.
	EMPTY = ""
)

var assertCount int32

func incrementAssertCount() {
	atomic.AddInt32(&assertCount, int32(1))
}

// Count returns the total number of assertions.
func Count() int {
	return int(assertCount)
}

// Predicate is a func that returns a bool.
type Predicate func(item interface{}) bool

//PredicateOfInt is a func that takes an int and returns a bool.
type PredicateOfInt func(item int) bool

// PredicateOfFloat is a func that takes a float64 and returns a bool.
type PredicateOfFloat func(item float64) bool

// PredicateOfString is a func that takes a string and returns a bool.
type PredicateOfString func(item string) bool

// PredicateOfTime is a func that takes a time.Time and returns a bool.
type PredicateOfTime func(item time.Time) bool

// Assertions is the main entry point for using the assertions library.
type Assertions struct {
	t           *testing.T
	didComplete bool
}

// Empty returns an empty assertions class; useful when you want to apply assertions w/o hooking into the testing framework.
func Empty() *Assertions {
	return &Assertions{}
}

// New returns a new instance of `Assertions`.
func New(t *testing.T) *Assertions {
	return &Assertions{t: t}
}

func (a *Assertions) assertion() {
	incrementAssertCount()
}

// NonFatal transitions the assertion into a `NonFatal` assertion; that is, one that will not cause the test to abort if it fails.
// NonFatal assertions are useful when you want to check many properties during a test, but only on an informational basis.
func (a *Assertions) NonFatal() *Optional { //golint you can bite me.
	return &Optional{a.t}
}

// NotNil asserts that a reference is not nil.
func (a *Assertions) NotNil(object interface{}, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldNotBeNil(object); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// Nil asserts that a reference is nil.
func (a *Assertions) Nil(object interface{}, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldBeNil(object); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// Len asserts that a collection has a given length.
func (a *Assertions) Len(collection interface{}, length int, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldHaveLength(collection, length); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// Empty asserts that a collection is empty.
func (a *Assertions) Empty(collection interface{}, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldBeEmpty(collection); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// NotEmpty asserts that a collection is not empty.
func (a *Assertions) NotEmpty(collection interface{}, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldNotBeEmpty(collection); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// Equal asserts that two objects are deeply equal.
func (a *Assertions) Equal(expected interface{}, actual interface{}, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldBeEqual(expected, actual); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// NotEqual asserts that two objects are not deeply equal.
func (a *Assertions) NotEqual(expected interface{}, actual interface{}, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldNotBeEqual(expected, actual); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// Zero asserts that a value is equal to it's default value.
func (a *Assertions) Zero(value interface{}, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldBeZero(value); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// NotZero asserts that a value is not equal to it's default value.
func (a *Assertions) NotZero(value interface{}, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldBeNonZero(value); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// True asserts a boolean is true.
func (a *Assertions) True(object bool, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldBeTrue(object); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// False asserts a boolean is false.
func (a *Assertions) False(object bool, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldBeFalse(object); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// InDelta asserts that two floats are within a delta.
func (a *Assertions) InDelta(f1, f2, delta float64, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldBeInDelta(f1, f2, delta); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// InTimeDelta asserts that times t1 and t2 are within a delta.
func (a *Assertions) InTimeDelta(t1, t2 time.Time, delta time.Duration, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldBeInTimeDelta(t1, t2, delta); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// FileExists asserts that a file exists at a given filepath on disk.
func (a *Assertions) FileExists(filepath string, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := fileShouldExist(filepath); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// Contains asserts that a substring is present in a corpus.
func (a *Assertions) Contains(substring, corpus string, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldContain(substring, corpus); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// Any applies a predicate.
func (a *Assertions) Any(target interface{}, predicate Predicate, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldAny(target, predicate); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// AnyOfInt applies a predicate.
func (a *Assertions) AnyOfInt(target []int, predicate PredicateOfInt, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldAnyOfInt(target, predicate); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// AnyOfFloat applies a predicate.
func (a *Assertions) AnyOfFloat(target []float64, predicate PredicateOfFloat, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldAnyOfFloat(target, predicate); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// AnyOfString applies a predicate.
func (a *Assertions) AnyOfString(target []string, predicate PredicateOfString, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldAnyOfString(target, predicate); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// All applies a predicate.
func (a *Assertions) All(target interface{}, predicate Predicate, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldAll(target, predicate); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// AllOfInt applies a predicate.
func (a *Assertions) AllOfInt(target []int, predicate PredicateOfInt, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldAllOfInt(target, predicate); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// AllOfFloat applies a predicate.
func (a *Assertions) AllOfFloat(target []float64, predicate PredicateOfFloat, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldAllOfFloat(target, predicate); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// AllOfString applies a predicate.
func (a *Assertions) AllOfString(target []string, predicate PredicateOfString, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldAllOfString(target, predicate); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// None applies a predicate.
func (a *Assertions) None(target interface{}, predicate Predicate, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldNone(target, predicate); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// NoneOfInt applies a predicate.
func (a *Assertions) NoneOfInt(target []int, predicate PredicateOfInt, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldNoneOfInt(target, predicate); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// NoneOfFloat applies a predicate.
func (a *Assertions) NoneOfFloat(target []float64, predicate PredicateOfFloat, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldNoneOfFloat(target, predicate); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// NoneOfString applies a predicate.
func (a *Assertions) NoneOfString(target []string, predicate PredicateOfString, userMessageComponents ...interface{}) {
	a.assertion()
	if didFail, message := shouldNoneOfString(target, predicate); didFail {
		failNow(a.t, message, userMessageComponents...)
	}
}

// FailNow forces a test failure (useful for debugging).
func (a *Assertions) FailNow(userMessageComponents ...interface{}) {
	failNow(a.t, "Fatal Assertion Failed", userMessageComponents...)
}

// StartTimeout starts a timed block.
func (a *Assertions) StartTimeout(timeout time.Duration, userMessageComponents ...interface{}) {
	sleepFor := 1 * time.Millisecond
	waited := time.Duration(0)
	a.didComplete = false

	go func() {
		for !a.didComplete {
			if waited > timeout {
				panic("Timeout Reached")
			}
			time.Sleep(sleepFor)
			waited += sleepFor
		}
	}()
}

// EndTimeout marks a timed block as complete.
func (a *Assertions) EndTimeout() {
	a.didComplete = true
}

// Optional is an assertion type that does not stop a test if an assertion fails, simply outputs the error.
type Optional struct {
	t *testing.T
}

func (o *Optional) assertion() {
	incrementAssertCount()
}

// Nil asserts the object is nil.
func (o *Optional) Nil(object interface{}, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldBeNil(object); didFail {
		fail(o.t, prefixOptional(message), userMessageComponents...)
		return false
	}
	return true
}

// NotNil asserts the object is not nil.
func (o *Optional) NotNil(object interface{}, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldNotBeNil(object); didFail {
		fail(o.t, prefixOptional(message), userMessageComponents...)
		return false
	}
	return true
}

// Len asserts that the collection has a specified length.
func (o *Optional) Len(collection interface{}, length int, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldHaveLength(collection, length); didFail {
		fail(o.t, prefixOptional(message), userMessageComponents...)
		return false
	}
	return true
}

// Empty asserts that a collection is empty.
func (o *Optional) Empty(collection interface{}, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldBeEmpty(collection); didFail {
		fail(o.t, prefixOptional(message), userMessageComponents...)
		return false
	}
	return true
}

// NotEmpty asserts that a collection is not empty.
func (o *Optional) NotEmpty(collection interface{}, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldNotBeEmpty(collection); didFail {
		fail(o.t, prefixOptional(message), userMessageComponents...)
		return false
	}
	return true
}

// Equal asserts that two objects are equal.
func (o *Optional) Equal(expected interface{}, actual interface{}, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldBeEqual(expected, actual); didFail {
		fail(o.t, prefixOptional(message), userMessageComponents...)
		return false
	}
	return true
}

// NotEqual asserts that two objects are not equal.
func (o *Optional) NotEqual(expected interface{}, actual interface{}, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldNotBeEqual(expected, actual); didFail {
		fail(o.t, prefixOptional(message), userMessageComponents...)
		return false
	}
	return true
}

// Zero asserts that a value is the default value.
func (o *Optional) Zero(value interface{}, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldBeZero(value); didFail {
		fail(o.t, prefixOptional(message), userMessageComponents...)
		return false
	}
	return true
}

// NotZero asserts that a value is not the default value.
func (o *Optional) NotZero(value interface{}, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldBeNonZero(value); didFail {
		fail(o.t, prefixOptional(message), userMessageComponents...)
		return false
	}
	return true
}

// True asserts that a bool is false.
func (o *Optional) True(object bool, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldBeTrue(object); didFail {
		fail(o.t, prefixOptional(message), userMessageComponents...)
		return false
	}
	return true
}

// False asserts that a bool is false.
func (o *Optional) False(object bool, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldBeFalse(object); didFail {
		fail(o.t, prefixOptional(message), userMessageComponents...)
		return false
	}
	return true
}

// InDelta returns if two float64s are separated by a given delta.
func (o *Optional) InDelta(a, b, delta float64, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldBeInDelta(a, b, delta); didFail {
		fail(o.t, prefixOptional(message), userMessageComponents...)
		return false
	}
	return true
}

// InTimeDelta returns if two times are separated by a given delta.
func (o *Optional) InTimeDelta(a, b time.Time, delta time.Duration, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldBeInTimeDelta(a, b, delta); didFail {
		fail(o.t, prefixOptional(message), userMessageComponents...)
		return false
	}
	return true
}

// FileExists asserts that a file exists on disk at a given filepath.
func (o *Optional) FileExists(filepath string, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := fileShouldExist(filepath); didFail {
		fail(o.t, prefixOptional(message), userMessageComponents...)
		return false
	}
	return true
}

// Contains checks if a substring is present in a corpus.
func (o *Optional) Contains(substring, corpus string, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldContain(substring, corpus); didFail {
		fail(o.t, prefixOptional(message), userMessageComponents...)
		return false
	}
	return true
}

// Any applies a predicate.
func (o *Optional) Any(target interface{}, predicate Predicate, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldAny(target, predicate); didFail {
		fail(o.t, prefixOptional(message), userMessageComponents...)
		return false
	}
	return true
}

// AnyOfInt applies a predicate.
func (o *Optional) AnyOfInt(target []int, predicate PredicateOfInt, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldAnyOfInt(target, predicate); didFail {
		failNow(o.t, message, userMessageComponents...)
		return false
	}
	return true
}

// AnyOfFloat applies a predicate.
func (o *Optional) AnyOfFloat(target []float64, predicate PredicateOfFloat, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldAnyOfFloat(target, predicate); didFail {
		failNow(o.t, message, userMessageComponents...)
		return false
	}
	return true
}

// AnyOfString applies a predicate.
func (o *Optional) AnyOfString(target []string, predicate PredicateOfString, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldAnyOfString(target, predicate); didFail {
		failNow(o.t, message, userMessageComponents...)
		return false
	}
	return true
}

// All applies a predicate.
func (o *Optional) All(target interface{}, predicate Predicate, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldAll(target, predicate); didFail {
		failNow(o.t, message, userMessageComponents...)
		return false
	}
	return true
}

// AllOfInt applies a predicate.
func (o *Optional) AllOfInt(target []int, predicate PredicateOfInt, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldAllOfInt(target, predicate); didFail {
		failNow(o.t, message, userMessageComponents...)
		return false
	}
	return true
}

// AllOfFloat applies a predicate.
func (o *Optional) AllOfFloat(target []float64, predicate PredicateOfFloat, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldAllOfFloat(target, predicate); didFail {
		failNow(o.t, message, userMessageComponents...)
		return false
	}
	return true
}

// AllOfString applies a predicate.
func (o *Optional) AllOfString(target []string, predicate PredicateOfString, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldAllOfString(target, predicate); didFail {
		failNow(o.t, message, userMessageComponents...)
		return false
	}
	return true
}

// None applies a predicate.
func (o *Optional) None(target interface{}, predicate Predicate, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldNone(target, predicate); didFail {
		failNow(o.t, message, userMessageComponents...)
		return false
	}
	return true
}

// NoneOfInt applies a predicate.
func (o *Optional) NoneOfInt(target []int, predicate PredicateOfInt, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldNoneOfInt(target, predicate); didFail {
		failNow(o.t, message, userMessageComponents...)
		return false
	}
	return true
}

// NoneOfFloat applies a predicate.
func (o *Optional) NoneOfFloat(target []float64, predicate PredicateOfFloat, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldNoneOfFloat(target, predicate); didFail {
		failNow(o.t, message, userMessageComponents...)
		return false
	}
	return true
}

// NoneOfString applies a predicate.
func (o *Optional) NoneOfString(target []string, predicate PredicateOfString, userMessageComponents ...interface{}) bool {
	o.assertion()
	if didFail, message := shouldNoneOfString(target, predicate); didFail {
		failNow(o.t, message, userMessageComponents...)
		return false
	}
	return true
}

// Fail manually injects a failure.
func (o *Optional) Fail(userMessageComponents ...interface{}) {
	fail(o.t, prefixOptional("Assertion Failed"), userMessageComponents...)
}

// --------------------------------------------------------------------------------
// OUTPUT
// --------------------------------------------------------------------------------

func failNow(t *testing.T, message string, userMessageComponents ...interface{}) {
	fail(t, message, userMessageComponents...)
	if t != nil {
		t.FailNow()
	} else {
		os.Exit(1)
	}
}

func fail(t *testing.T, message string, userMessageComponents ...interface{}) {
	errorTrace := strings.Join(callerInfo(), "\n\t")

	if len(errorTrace) == 0 {
		errorTrace = "Unknown"
	}

	assertionFailedLabel := color("Assertion Failed!", RED)
	locationLabel := color("Assert Location", GRAY)
	assertionLabel := color("Assertion", GRAY)
	messageLabel := color("Message", GRAY)

	erasure := fmt.Sprintf("\r%s", getClearString())

	if len(userMessageComponents) != 0 {
		userMessage := fmt.Sprint(userMessageComponents...)

		errorFormat := `%s
%s
%s:
	%s
%s: 
	%s
%s: 
	%s

`
		if t != nil {
			t.Errorf(errorFormat, erasure, assertionFailedLabel, locationLabel, errorTrace, assertionLabel, message, messageLabel, userMessage)
		} else {
			fmt.Fprintf(os.Stderr, errorFormat, "", assertionFailedLabel, locationLabel, errorTrace, assertionLabel, message, messageLabel, userMessage)
		}

	} else {
		errorFormat := `%s
%s
%s: 
	%s
%s: 
	%s

`
		if t != nil {
			t.Errorf(errorFormat, erasure, assertionFailedLabel, locationLabel, errorTrace, assertionLabel, message)
		} else {
			fmt.Fprintf(os.Stderr, errorFormat, "", assertionFailedLabel, locationLabel, errorTrace, assertionLabel, message)
		}
	}
}

// --------------------------------------------------------------------------------
// ASSERTION LOGIC
// --------------------------------------------------------------------------------

func shouldHaveLength(collection interface{}, length int) (bool, string) {
	if l := getLength(collection); l != length {
		message := shouldBeMultipleMessage(length, l, "Collection should have length")
		return true, message
	}
	return false, EMPTY
}

func shouldNotBeEmpty(collection interface{}) (bool, string) {
	if l := getLength(collection); l == 0 {
		message := "Should not be empty"
		return true, message
	}
	return false, EMPTY
}

func shouldBeEmpty(collection interface{}) (bool, string) {
	if l := getLength(collection); l != 0 {
		message := shouldBeMessage(collection, "Should be empty")
		return true, message
	}
	return false, EMPTY
}

func shouldBeEqual(expected, actual interface{}) (bool, string) {
	if !areEqual(expected, actual) {
		return true, equalMessage(actual, expected)
	}
	return false, EMPTY
}

func shouldNotBeEqual(expected, actual interface{}) (bool, string) {
	if areEqual(expected, actual) {
		return true, notEqualMessage(actual, expected)
	}
	return false, EMPTY
}

func shouldNotBeNil(object interface{}) (bool, string) {
	if isNil(object) {
		return true, "Should not be nil"
	}
	return false, EMPTY
}

func shouldBeNil(object interface{}) (bool, string) {
	if !isNil(object) {
		return true, shouldBeMessage(object, "Should be nil")
	}
	return false, EMPTY
}

func shouldBeTrue(value bool) (bool, string) {
	if !value {
		return true, "Should be true"
	}
	return false, EMPTY
}

func shouldBeFalse(value bool) (bool, string) {
	if value {
		return true, "Should be false"
	}
	return false, EMPTY
}

func shouldBeZero(value interface{}) (bool, string) {
	if !isZero(value) {
		return true, shouldBeMessage(value, "Should be zero")
	}
	return false, EMPTY
}

func shouldBeNonZero(value interface{}) (bool, string) {
	if isZero(value) {
		return true, "Should be non-zero"
	}
	return false, EMPTY
}

func fileShouldExist(filePath string) (bool, string) {
	_, err := os.Stat(filePath)
	if err != nil {
		pwd, _ := os.Getwd()
		message := fmt.Sprintf("File doesnt exist: %s, `pwd`: %s", filePath, pwd)
		return true, message
	}
	return false, EMPTY
}

func shouldBeInDelta(from, to, delta float64) (bool, string) {
	diff := math.Abs(from - to)
	if diff > delta {
		message := fmt.Sprintf("Difference of %0.5f and %0.5f should be less than %0.5f", from, to, delta)
		return true, message
	}
	return false, EMPTY
}

func shouldBeInTimeDelta(from, to time.Time, delta time.Duration) (bool, string) {
	var diff time.Duration
	if from.After(to) {
		diff = from.Sub(to)
	} else {
		diff = to.Sub(from)
	}
	if diff > delta {
		message := fmt.Sprintf("Delta of %s and %s should be less than %v", from.Format(time.RFC3339), to.Format(time.RFC3339), delta)
		return true, message
	}
	return false, EMPTY
}

func shouldContain(subString, corpus string) (bool, string) {
	if !strings.Contains(corpus, subString) {
		message := fmt.Sprintf("`%s` should contain `%s`", corpus, subString)
		return true, message
	}
	return false, EMPTY
}

func shouldAny(target interface{}, predicate Predicate) (bool, string) {
	t := reflect.TypeOf(target)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	v := reflect.ValueOf(target)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if t.Kind() != reflect.Slice {
		return true, "`target` is not a slice"
	}

	for x := 0; x < v.Len(); x++ {
		obj := v.Index(x).Interface()
		if predicate(obj) {
			return false, EMPTY
		}
	}
	return true, "Predicate did not fire for any element in target"
}

func shouldAnyOfInt(target []int, predicate PredicateOfInt) (bool, string) {
	v := reflect.ValueOf(target)

	for x := 0; x < v.Len(); x++ {
		obj := v.Index(x).Interface().(int)
		if predicate(obj) {
			return false, EMPTY
		}
	}
	return true, "Predicate did not fire for any element in target"
}

func shouldAnyOfFloat(target []float64, predicate PredicateOfFloat) (bool, string) {
	v := reflect.ValueOf(target)

	for x := 0; x < v.Len(); x++ {
		obj := v.Index(x).Interface().(float64)
		if predicate(obj) {
			return false, EMPTY
		}
	}
	return true, "Predicate did not fire for any element in target"
}

func shouldAnyOfString(target []string, predicate PredicateOfString) (bool, string) {
	v := reflect.ValueOf(target)

	for x := 0; x < v.Len(); x++ {
		obj := v.Index(x).Interface().(string)
		if predicate(obj) {
			return false, EMPTY
		}
	}
	return true, "Predicate did not fire for any element in target"
}

func shouldAll(target interface{}, predicate Predicate) (bool, string) {
	t := reflect.TypeOf(target)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	v := reflect.ValueOf(target)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if t.Kind() != reflect.Slice {
		return true, "`target` is not a slice"
	}

	for x := 0; x < v.Len(); x++ {
		obj := v.Index(x).Interface()
		if !predicate(obj) {
			return true, fmt.Sprintf("Predicate failed for element in target: %#v", obj)
		}
	}
	return false, EMPTY
}

func shouldAllOfInt(target []int, predicate PredicateOfInt) (bool, string) {
	v := reflect.ValueOf(target)

	for x := 0; x < v.Len(); x++ {
		obj := v.Index(x).Interface().(int)
		if !predicate(obj) {
			return true, fmt.Sprintf("Predicate failed for element in target: %#v", obj)
		}
	}
	return false, EMPTY
}

func shouldAllOfFloat(target []float64, predicate PredicateOfFloat) (bool, string) {
	v := reflect.ValueOf(target)

	for x := 0; x < v.Len(); x++ {
		obj := v.Index(x).Interface().(float64)
		if !predicate(obj) {
			return true, fmt.Sprintf("Predicate failed for element in target: %#v", obj)
		}
	}
	return false, EMPTY
}

func shouldAllOfString(target []string, predicate PredicateOfString) (bool, string) {
	v := reflect.ValueOf(target)

	for x := 0; x < v.Len(); x++ {
		obj := v.Index(x).Interface().(string)
		if !predicate(obj) {
			return true, fmt.Sprintf("Predicate failed for element in target: %#v", obj)
		}
	}
	return false, EMPTY
}

func shouldNone(target interface{}, predicate Predicate) (bool, string) {
	t := reflect.TypeOf(target)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	v := reflect.ValueOf(target)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if t.Kind() != reflect.Slice {
		return true, "`target` is not a slice"
	}

	for x := 0; x < v.Len(); x++ {
		obj := v.Index(x).Interface()
		if predicate(obj) {
			return true, fmt.Sprintf("Predicate passed for element in target: %#v", obj)
		}
	}
	return false, EMPTY
}

func shouldNoneOfInt(target []int, predicate PredicateOfInt) (bool, string) {
	v := reflect.ValueOf(target)

	for x := 0; x < v.Len(); x++ {
		obj := v.Index(x).Interface().(int)
		if predicate(obj) {
			return true, fmt.Sprintf("Predicate passed for element in target: %#v", obj)
		}
	}
	return false, EMPTY
}

func shouldNoneOfFloat(target []float64, predicate PredicateOfFloat) (bool, string) {
	v := reflect.ValueOf(target)

	for x := 0; x < v.Len(); x++ {
		obj := v.Index(x).Interface().(float64)
		if predicate(obj) {
			return true, fmt.Sprintf("Predicate passed for element in target: %#v", obj)
		}
	}
	return false, EMPTY
}

func shouldNoneOfString(target []string, predicate PredicateOfString) (bool, string) {
	v := reflect.ValueOf(target)

	for x := 0; x < v.Len(); x++ {
		obj := v.Index(x).Interface().(string)
		if predicate(obj) {
			return true, fmt.Sprintf("Predicate passed for element in target: %#v", obj)
		}
	}
	return false, EMPTY
}

// --------------------------------------------------------------------------------
// UTILITY
// --------------------------------------------------------------------------------

func prefixOptional(message string) string {
	return "(Non-Fatal) " + message
}

func shouldBeMultipleMessage(expected, actual interface{}, message string) string {
	expectedLabel := color("Expected", WHITE)
	actualLabel := color("Actual", WHITE)

	return fmt.Sprintf(`%s
	%s: 	%v
	%s: 	%v`, message, expectedLabel, expected, actualLabel, actual)
}

func shouldBeMessage(object interface{}, message string) string {
	actualLabel := color("Actual", WHITE)
	return fmt.Sprintf(`%s
	%s: 	%v`, message, actualLabel, object)
}

func notEqualMessage(actual, expected interface{}) string {
	return shouldBeMultipleMessage(expected, actual, "Objects should not be equal")
}

func equalMessage(actual, expected interface{}) string {
	return shouldBeMultipleMessage(expected, actual, "Objects should be equal")
}

func getLength(object interface{}) int {
	if object == nil {
		return 0
	} else if object == "" {
		return 0
	}

	objValue := reflect.ValueOf(object)

	switch objValue.Kind() {
	case reflect.Map:
		fallthrough
	case reflect.Slice, reflect.Chan, reflect.String:
		{
			return objValue.Len()
		}
	}
	return 0
}

func isNil(object interface{}) bool {
	if object == nil {
		return true
	}

	value := reflect.ValueOf(object)
	kind := value.Kind()
	if kind >= reflect.Chan && kind <= reflect.Slice && value.IsNil() {
		return true
	}
	return false
}

func isZero(value interface{}) bool {
	return areEqual(0, value)
}

func areEqual(expected, actual interface{}) bool {
	if expected == nil && actual == nil {
		return true
	}
	if (expected == nil && actual != nil) || (expected != nil && actual == nil) {
		return false
	}

	actualType := reflect.TypeOf(actual)
	if actualType == nil {
		return false
	}
	expectedValue := reflect.ValueOf(expected)
	if expectedValue.IsValid() && expectedValue.Type().ConvertibleTo(actualType) {
		return reflect.DeepEqual(expectedValue.Convert(actualType).Interface(), actual)
	}

	return reflect.DeepEqual(expected, actual)
}

func callerInfo() []string {
	pc := uintptr(0)
	file := ""
	line := 0
	ok := false
	name := ""

	callers := []string{}
	for i := 0; ; i++ {
		pc, file, line, ok = runtime.Caller(i)
		if !ok {
			return nil
		}

		if file == "<autogenerated>" {
			break
		}

		parts := strings.Split(file, "/")
		dir := parts[len(parts)-2]
		file = parts[len(parts)-1]
		if dir != "assert" && dir != "go-assert" && dir != "mock" && dir != "require" {
			callers = append(callers, fmt.Sprintf("%s:%d", file, line))
		}

		f := runtime.FuncForPC(pc)
		if f == nil {
			break
		}
		name = f.Name()

		// Drop the package
		segments := strings.Split(name, ".")
		name = segments[len(segments)-1]
		if isTest(name, "Test") ||
			isTest(name, "Benchmark") ||
			isTest(name, "Example") {
			break
		}
	}

	return callers
}

func color(input string, colorCode string) string {
	return fmt.Sprintf("\033[%s;01m%s\033[0m", colorCode, input)
}

func reflectTypeName(object interface{}) string {
	return reflect.TypeOf(object).Name()
}

func isTest(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	if len(name) == len(prefix) { // "Test" is ok
		return true
	}
	rune, _ := utf8.DecodeRuneInString(name[len(prefix):])
	return !unicode.IsLower(rune)
}

func getClearString() string {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		return ""
	}
	parts := strings.Split(file, "/")
	file = parts[len(parts)-1]

	return strings.Repeat(" ", len(fmt.Sprintf("%s:%d:      ", file, line))+2)
}
