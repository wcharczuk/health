package workqueue

import (
	"fmt"

	exception "github.com/blendlabs/go-exception"
)

// Entry is an individual item of work.
type Entry struct {
	Action Action
	Args   []interface{}
	Tries  int32
}

func (e Entry) String() string {
	return fmt.Sprintf("{ %#v args: %v tries: %d }", e.Action, e.Args, e.Tries)
}

// Execute runs the work item.
func (e Entry) Execute() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = exception.Nest(err, fmt.Errorf("%v", r))
		}
	}()

	err = e.Action(e.Args...)
	return
}
