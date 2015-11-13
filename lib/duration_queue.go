package lib

import (
	"errors"
	"math"
	"sort"
	"time"
)

type durationList []time.Duration

func (dl durationList) Len() int {
	return len(dl)
}

func (dl durationList) Less(i, j int) bool {
	return dl[i] < dl[j]
}

func (dl durationList) Swap(i, j int) {
	dl[i], dl[j] = dl[j], dl[i]
}

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

func (dq *DurationQueue) ToArray() []time.Duration {
	if dq.Head == nil {
		return []time.Duration{}
	}

	results := []time.Duration{}
	node_ptr := dq.Head
	for node_ptr != nil {
		results = append(results, node_ptr.Value)
		node_ptr = node_ptr.Previous
	}

	return results
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

func (dq *DurationQueue) Percentile(percentile float64) time.Duration {
	if dq.Head == nil {
		return time.Duration(0)
	}

	values := dq.ToArray()
	sort.Sort(durationList(values))

	index := (percentile / 100.0) * float64(len(values))
	if index == float64(int64(index)) {
		i := float64ToInt(index)

		if i < 1 {
			return time.Duration(0)
		}

		value_1 := float64(values[i-1])
		value_2 := float64(values[i])
		to_average := []float64{value_1, value_2}
		averaged := mean(to_average)

		return time.Duration(int64(averaged))
	} else {
		i := float64ToInt(index)
		if i < 1 {
			return time.Duration(0)
		}

		return values[i-1]
	}
}

func mean(input []float64) float64 {
	accum := 0.0
	input_len := len(input)
	for i := 0; i < input_len; i++ {
		v := input[i]
		accum = accum + float64(v)
	}
	return accum / float64(input_len)
}

func round(input float64, places int) (rounded float64, err error) {
	if math.IsNaN(input) {
		return 0.0, errors.New("Not a number")
	}

	sign := 1.0
	if input < 0 {
		sign = -1
		input *= -1
	}

	precision := math.Pow(10, float64(places))
	digit := input * precision
	_, decimal := math.Modf(digit)

	if decimal >= 0.5 {
		rounded = math.Ceil(digit)
	} else {
		rounded = math.Floor(digit)
	}

	return rounded / precision * sign, nil
}

func float64ToInt(input float64) (output int) {
	r, _ := round(input, 0)
	return int(r)
}
