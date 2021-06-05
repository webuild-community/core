package queue

import (
	"container/list"
)

type queue struct {
	isConsuming bool
	l           *list.List
}

// NewQueueService --
func NewQueueService() Service {
	return &queue{l: list.New()}
}

func (q *queue) Add(value interface{}) error {
	q.l.PushFront(value)
	return nil
}

func (q *queue) Consume() interface{} {
	e := q.l.Front()
	if e == nil {
		return nil
	}

	val := e.Value
	q.l.Remove(e)
	return val
}

func (q *queue) SetIsConsuming(c bool) {
	q.isConsuming = c
}

func (q *queue) GetIsConsuming() bool {
	return q.isConsuming
}
