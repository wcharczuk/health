package workqueue

import "sync/atomic"

// NewWorker creates a new worker.
func NewWorker(id int, parent *Queue, maxItems int) *Worker {
	return &Worker{
		ID:     id,
		Parent: parent,

		Work:  make(chan *Entry, maxItems),
		Abort: make(chan bool),
	}
}

// Worker is a consumer of the work queue.
type Worker struct {
	ID     int
	Work   chan *Entry
	Parent *Queue
	Abort  chan bool
}

// Start starts the worker.
func (w *Worker) Start() {
	go processWork(w.Work, w.Parent, w.Abort)
}

func processWork(work chan *Entry, parent *Queue, abort chan bool) {
	var err error
	var workItem *Entry
	for {
		select {
		case workItem = <-work:
			if workItem == nil {
				continue
			}
			err = workItem.Execute()
			if err != nil {
				atomic.AddInt32(&workItem.Tries, 1)
				if workItem.Tries < int32(parent.maxRetries) {
					parent.actionQueue <- workItem
					continue
				}
			}
			parent.entryPool.Put(workItem)
		case <-abort:
			return
		}
	}
}

// Close sends the stop signal to the worker.
func (w *Worker) Close() error {
	w.Abort <- true
	close(w.Abort)
	close(w.Work)

	w.Abort = nil
	w.Work = nil
	w.Parent = nil
	return nil
}
