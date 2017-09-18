Work Queue
=============

[![Build Status](https://travis-ci.org/wcharczuk/go-workqueue.svg?branch=master)](https://travis-ci.org/wcharczuk/go-workqueue)

This library implements a simple work queue, driven by a configurable number of goroutine workers.

##Example

```golang
wq := workqueue.New()
workQueue.Start() //two workers, i.e. two go routines
workQueue.Enqueue(func(v ...interface{}) error {
    fmt.Printf("Work Item: %#v\n", v)
    return nil
}, "hello world")
// output: "Work Item: hello world"
```