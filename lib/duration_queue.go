package lib

import (
	"math"
	"time"
)

type durationNode struct {
	Value    time.Duration
	Previous *durationNode
	Next     *durationNode
}

type DurationQueue struct {
	Head   *durationNode
	Tail   *durationNode
	Length int
}

func (dq *DurationQueue) Push(value time.Duration) {
	new_node := durationNode{Value: value}
	new_node.Next = dq.Tail

	if dq.Tail != nil {
		dq.Tail.Previous = &new_node
	}

	if dq.Head == nil {
		dq.Head = &new_node
	}

	dq.Tail = &new_node
	dq.Length = dq.Length + 1
}

func (dq *DurationQueue) Pop() *time.Duration {
	if dq.Head == nil {
		return nil
	}

	old_head := dq.Head
	value := old_head.Value

	dq.Head = dq.Head.Previous
	if dq.Head == nil {
		dq.Tail = nil
	}

	dq.Length = dq.Length - 1

	return &value
}

func (dq *DurationQueue) Peek() *time.Duration {
	if dq.Head == nil {
		return nil
	}

	return &dq.Head.Value
}

func (dq *DurationQueue) PeekBack() *time.Duration {
	if dq.Tail == nil {
		return nil
	}

	return &dq.Tail.Value
}

func (dq *DurationQueue) Mean() time.Duration {
	if dq.Head == nil {
		return 0
	}

	accum := time.Duration(0)

	node_ptr := dq.Head
	for node_ptr != nil {
		accum = accum + node_ptr.Value
		node_ptr = node_ptr.Previous
	}

	return accum / time.Duration(dq.Length)
}

// Population variance
func (dq *DurationQueue) Variance() time.Duration {
	variance := 0.0

	if dq.Head == nil {
		return 0
	}

	m := float64(dq.Mean())

	node_ptr := dq.Head
	for node_ptr != nil {
		n := float64(node_ptr.Value)

		diff := n - m
		squared := diff * diff

		variance = variance + squared
		node_ptr = node_ptr.Previous
	}

	v := variance / float64(dq.Length)
	return time.Duration(v)
}

func (dq *DurationQueue) StdDev() time.Duration {
	vp := float64(dq.Variance())
	sp := math.Pow(vp, 0.5)
	return time.Duration(sp)
}
