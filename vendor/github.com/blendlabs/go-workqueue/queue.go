package workqueue

import (
	"bytes"
	"fmt"
	"runtime"
	"sync"
)

const (
	// DefaultMaxRetries is the maximum times a process queue item will be retried before being dropped.
	DefaultMaxRetries = 10

	// DefaultMaxWorkItems is the default entry buffer length.
	// Currently the default is 2^18 or 256k.
	// WorkItems maps to the initialized capacity of a buffered channel.
	// As a result it does not reflect actual memory consumed.
	DefaultMaxWorkItems = 1 << 18
)

var (
	_default     *Queue
	_defaultLock sync.Mutex
)

// Default returns a singleton queue.
func Default() *Queue {
	if _default == nil {
		_defaultLock.Lock()
		defer _defaultLock.Unlock()
		if _default == nil {
			_default = New()
		}
	}
	return _default
}

// Action is an action that can be dispatched by the process queue.
type Action func(args ...interface{}) error

// New returns a new work queue.
func New() *Queue {
	return &Queue{
		numWorkers:   runtime.NumCPU(),
		maxRetries:   DefaultMaxRetries,
		maxWorkItems: DefaultMaxWorkItems,
	}
}

// NewWithWorkers returns a new work queue with a given number of workers.
func NewWithWorkers(numWorkers int) *Queue {
	return &Queue{
		numWorkers:   numWorkers,
		maxRetries:   DefaultMaxRetries,
		maxWorkItems: DefaultMaxWorkItems,
	}
}

// NewWithOptions returns a new queue with customizable options.
func NewWithOptions(numWorkers, retryCount, maxWorkItems int) *Queue {
	return &Queue{
		numWorkers:   numWorkers,
		maxRetries:   retryCount,
		maxWorkItems: maxWorkItems,
	}
}

// Queue is the container for work items, it dispatches work to the workers.
type Queue struct {
	numWorkers   int
	maxRetries   int
	maxWorkItems int

	running bool

	actionQueue chan *Entry

	entryPool   sync.Pool
	workers     []*Worker
	abortSignal chan bool
}

// Start starts the dispatcher workers for the process quere.
func (q *Queue) Start() {
	if q.running {
		return
	}

	q.workers = make([]*Worker, q.numWorkers)
	q.actionQueue = make(chan *Entry, q.maxWorkItems)
	q.abortSignal = make(chan bool)
	q.entryPool = sync.Pool{
		New: func() interface{} {
			return &Entry{}
		},
	}
	q.running = true

	for id := 0; id < q.numWorkers; id++ {
		q.newWorker(id)
	}

	go dispatch(q.workers, q.actionQueue, q.abortSignal)
}

// Len returns the number of items in the work queue.
func (q *Queue) Len() int {
	return len(q.actionQueue)
}

// NumWorkers returns the number of worker routines.
func (q *Queue) NumWorkers() int {
	return q.numWorkers
}

// SetNumWorkers lets you set the num workers.
func (q *Queue) SetNumWorkers(workers int) {
	q.numWorkers = workers
	if q.running {
		q.Close()
		q.Start()
	}
}

// MaxWorkItems returns the maximum length of the work item queue.
func (q *Queue) MaxWorkItems() int {
	return q.maxWorkItems
}

// SetMaxWorkItems sets the max work items.
func (q *Queue) SetMaxWorkItems(workItems int) {
	q.maxWorkItems = workItems
	if q.running {
		q.Close()
		q.Start()
	}
}

// MaxRetries returns the maximum number of retries.
func (q *Queue) MaxRetries() int {
	return q.maxRetries
}

// SetMaxRetries sets the maximum nummer of retries for a work item on error.
func (q *Queue) SetMaxRetries(maxRetries int) {
	q.maxRetries = maxRetries
}

// Running returns if the queue has started or not.
func (q *Queue) Running() bool {
	return q.running
}

// Enqueue adds a work item to the process queue.
func (q *Queue) Enqueue(action Action, args ...interface{}) {
	if !q.running {
		return
	}
	entry := q.entryPool.Get().(*Entry)
	entry.Action = action
	entry.Args = args
	entry.Tries = 0
	q.actionQueue <- entry
}

// Close drains the queue and stops the workers.
func (q *Queue) Close() error {
	if !q.running {
		return nil
	}

	q.abortSignal <- true

	close(q.abortSignal)
	close(q.actionQueue)

	var err error
	for x := 0; x < len(q.workers); x++ {
		err = q.workers[x].Close()
		if err != nil {
			return err
		}
	}

	q.workers = nil
	q.actionQueue = nil
	q.running = false
	return nil
}

// String returns a string representation of the queue.
func (q *Queue) String() string {
	b := bytes.NewBuffer([]byte{})
	b.WriteString(fmt.Sprintf("WorkQueue [%d]", q.Len()))
	if q.Len() > 0 {
		q.Each(func(e *Entry) {
			b.WriteString(" ")
			b.WriteString(e.String())
		})
	}
	return b.String()
}

// Each runs the consumer for each item in the queue.
func (q *Queue) Each(visitor func(entry *Entry)) {
	queueLength := len(q.actionQueue)
	var entry *Entry
	for x := 0; x < queueLength; x++ {
		entry = <-q.actionQueue
		visitor(entry)
		q.actionQueue <- entry
	}
}

func (q *Queue) newWorker(id int) {
	q.workers[id] = NewWorker(id, q, q.maxWorkItems/q.numWorkers)
	q.workers[id].Start()
}

func dispatch(workers []*Worker, work chan *Entry, abort chan bool) {
	var workItem *Entry
	var workerIndex int
	numWorkers := len(workers)
	for {
		select {
		case workItem = <-work:
			if workItem == nil {
				continue
			}
			workers[workerIndex].Work <- workItem
			if numWorkers > 1 {
				workerIndex++
				if workerIndex >= numWorkers {
					workerIndex = 0
				}
			}
		case <-abort:
			return
		}
	}
}
