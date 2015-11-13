package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBasicOperations(t *testing.T) {
	assert := assert.New(t)
	queue := &DurationQueue{}

	assert.Nil(queue.Head)
	assert.Nil(queue.Tail)

	queue.Push(time.Duration(1 * time.Second))

	assert.NotNil(queue.Head)
	assert.NotNil(queue.Tail)
	assert.Equal(1, queue.Length)

	queue.Push(time.Duration(1500 * time.Millisecond))

	assert.NotNil(queue.Head)
	assert.NotNil(queue.Head.Previous)
	assert.NotNil(queue.Tail)
	assert.Equal(2, queue.Length)

	queue.Push(time.Duration(2 * time.Second))

	assert.Equal(3, queue.Length)
	assert.NotNil(queue.Head)
	assert.NotNil(queue.Head.Previous)
	assert.NotNil(queue.Head.Previous.Previous)
	assert.NotNil(queue.Tail)

	first_peeked_value := queue.Peek()
	assert.Equal(1*time.Second, *first_peeked_value)
	assert.Equal(3, queue.Length)
	assert.NotNil(queue.Head)
	assert.NotNil(queue.Head.Previous)
	assert.NotNil(queue.Head.Previous.Previous)
	assert.NotNil(queue.Tail)

	first_value := queue.Pop()

	assert.NotNil(first_value)
	assert.Equal(1*time.Second, *first_value)
	assert.Equal(2, queue.Length)
	assert.NotNil(queue.Head)
	assert.NotNil(queue.Head.Previous)
	assert.NotNil(queue.Tail)

	second_value := queue.Pop()
	assert.NotNil(second_value)
	assert.Equal(1500*time.Millisecond, *second_value)
	assert.Equal(1, queue.Length)
	assert.NotNil(queue.Head)
	assert.NotNil(queue.Tail)
	assert.Nil(queue.Head.Previous)

	third_value := queue.Pop()
	assert.NotNil(third_value)
	assert.Equal(2*time.Second, *third_value)
	assert.Equal(0, queue.Length)
	assert.Nil(queue.Head)
	assert.Nil(queue.Tail)

	nil_value := queue.Pop()
	assert.Nil(nil_value)
	assert.Zero(queue.Length)

	another_nil_value := queue.Peek()
	assert.Nil(another_nil_value)
	assert.Zero(queue.Length)
}

func TestStats(t *testing.T) {
	assert := assert.New(t)
	queue := &DurationQueue{}
	queue.Push(1 * time.Second)
	queue.Push(1 * time.Second)
	queue.Push(1 * time.Second)
	queue.Push(1 * time.Second)
	queue.Push(1 * time.Second)

	m := queue.Mean()
	v := queue.Variance()
	s := queue.StdDev()

	assert.Equal(1*time.Second, m)
	assert.Zero(v)
	assert.Zero(s)
}

func TestStatsAdvanced(t *testing.T) {
	assert := assert.New(t)
	queue := &DurationQueue{}
	queue.Push(1 * time.Second)
	queue.Push(2 * time.Second)
	queue.Push(3 * time.Second)
	queue.Push(4 * time.Second)
	queue.Push(5 * time.Second)

	m := queue.Mean()

	assert.Equal(3*time.Second, m)
}

func TestStatsAdvancedIntense(t *testing.T) {
	assert := assert.New(t)
	queue := &DurationQueue{}

	for i := 0; i < 1000; i++ {
		queue.Push(time.Duration(i) * time.Second)
	}

	m := queue.Mean()
	assert.NotZero(m)
	assert.True(m > 0)
}
