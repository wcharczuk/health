package logger

import (
	"os"
	"strings"
)

// NewEventSet returns a new EventSet with the given events enabled.
func NewEventSet(eventFlags ...Event) *EventSet {
	efs := &EventSet{
		flags: make(map[Event]bool),
	}
	for _, flag := range eventFlags {
		efs.Enable(flag)
	}
	return efs
}

// NewEventSetAll returns a new EventSet with all flags enabled.
func NewEventSetAll() *EventSet {
	return &EventSet{
		flags: make(map[Event]bool),
		all:   true,
	}
}

// NewEventSetNone returns a new EventSet with no flags enabled.
func NewEventSetNone() *EventSet {
	return &EventSet{
		flags: make(map[Event]bool),
		none:  true,
	}
}

// NewEventSetFromEnv returns a new EventSet from the environment.
func NewEventSetFromEnv() *EventSet {
	envEventsFlag := os.Getenv(EnvironmentVariableLogEvents)
	if len(envEventsFlag) > 0 {
		return NewEventSetFromCSV(envEventsFlag)
	}
	return NewEventSet()
}

// NewEventSetFromCSV returns a new event flag set from a csv of event flags.
// These flags are case insensitive.
func NewEventSetFromCSV(flagCSV string) *EventSet {
	flagSet := &EventSet{
		flags: map[Event]bool{},
	}

	flags := strings.Split(flagCSV, ",")

	for _, flag := range flags {
		parsedFlag := Event(strings.Trim(strings.ToLower(flag), " \t\n"))
		if string(parsedFlag) == string(EventAll) {
			flagSet.all = true
		}

		if string(parsedFlag) == string(EventNone) {
			flagSet.none = true
		}

		if strings.HasPrefix(string(parsedFlag), "-") {
			flag := Event(strings.TrimPrefix(string(parsedFlag), "-"))
			flagSet.flags[flag] = false
		} else {
			flagSet.flags[parsedFlag] = true
		}
	}

	return flagSet
}

// EventSet is a set of event flags.
type EventSet struct {
	flags map[Event]bool
	all   bool
	none  bool
}

// Enable enables an event flag.
func (efs *EventSet) Enable(flagValue Event) {
	efs.none = false
	efs.flags[flagValue] = true
}

// Disable disabled an event flag.
func (efs *EventSet) Disable(flagValue Event) {
	efs.flags[flagValue] = false
}

// EnableAll flips the `all` bit on the flag set.
func (efs *EventSet) EnableAll() {
	efs.all = true
	efs.none = false
}

// IsAllEnabled returns if the all bit is flipped on.
func (efs *EventSet) IsAllEnabled() bool {
	return efs.all
}

// IsNoneEnabled returns if the none bit is flipped on.
func (efs *EventSet) IsNoneEnabled() bool {
	return efs.none
}

// DisableAll flips the `none` bit on the flag set.
func (efs *EventSet) DisableAll() {
	efs.all = false
	efs.none = true
}

// IsEnabled checks to see if an event is enabled.
func (efs EventSet) IsEnabled(flagValue Event) bool {
	if efs.all {
		// figure out if we explicitly disabled the flag.
		if enabled, hasEvent := efs.flags[flagValue]; hasEvent && !enabled {
			return false
		}
		return true
	}
	if efs.none {
		return false
	}
	if efs.flags != nil {
		if enabled, hasFlag := efs.flags[flagValue]; hasFlag {
			return enabled
		}
	}
	return false
}

func (efs EventSet) String() string {
	if efs.none {
		return string(EventNone)
	}

	var flags []string
	if efs.all {
		flags = []string{string(EventAll)}
	}
	for key, enabled := range efs.flags {
		if key != EventAll {
			if enabled {
				flags = append(flags, string(key))
			} else {
				flags = append(flags, "-"+string(key))
			}
		}
	}
	return strings.Join(flags, ", ")
}

// CoalesceWith sets the set from another, with the other taking precedence.
func (efs *EventSet) CoalesceWith(other *EventSet) {
	if other.all {
		efs.all = true
	}
	if other.none {
		efs.none = true
	}
	for key, value := range other.flags {
		efs.flags[key] = value
	}
}
