package queue

import (
	"fmt"
	"reflect"
)

// SliceQueue ye
type SliceQueue struct {
	data      []interface{}
	queueType reflect.Type
}

func newSliceQueue(data ...interface{}) (q *SliceQueue, err error) {
	if len(data) == 0 {
		err = fmt.Errorf("No data given to SliceQueue")
		return nil, err
	}
	q = &SliceQueue{queueType: reflect.TypeOf(data[0])}

	for _, v := range data {
		err = q.push(v)
		if err != nil {
			return nil, err
		}
	}
	return q, nil
}

func (q *SliceQueue) push(data interface{}) (err error) {
	if reflect.TypeOf(data) != q.queueType {
		err = fmt.Errorf("Data not of consistent type:\nWanted:%v\nGot:%v", q.queueType, reflect.TypeOf(data))
		return err
	}
	q.data = append(q.data, data)
	return nil
}

func (q *SliceQueue) pop() (data interface{}, err error) {
	if len(q.data) == 0 {
		err = fmt.Errorf("No data in queue")
		return nil, err
	}
	data = q.data[0]
	q.data = q.data[1:]
	return data, nil
}

func (q SliceQueue) peek() (data interface{}) {
	if len(q.data) == 0 {
		return nil
	}
	data = q.data[0]
	return data
}

// ListQueue - It's a queue with linked lists dawg
type ListQueue struct {
	head      *node
	tail      *node
	queueType reflect.Type
}

type node struct {
	next *node
	data interface{}
}

func newListQueue(data ...interface{}) (q *ListQueue, err error) {
	if len(data) == 0 {
		err = fmt.Errorf("No data given to ListQueue")
		return nil, err
	}
	q = &ListQueue{queueType: reflect.TypeOf(data[0])}

	for _, v := range data {
		err = q.push(v)
		if err != nil {
			return nil, err
		}
	}
	return q, nil
}

func (q *ListQueue) push(data interface{}) (err error) {
	if reflect.TypeOf(data) != q.queueType {
		err = fmt.Errorf("Data not of consistent type:\nWanted:%v\nGot:%v", q.queueType, reflect.TypeOf(data))
		return err
	}
	node := new(node)
	node.data = data

	if q.head == nil {
		q.head = node
		q.tail = node
	} else {
		q.tail.next = node
		q.tail = node
	}
	return nil
}

func (q *ListQueue) pop() (data interface{}, err error) {
	if q.head == nil {
		err = fmt.Errorf("No data in queue")
		return nil, err
	}
	data = q.head.data
	q.head = q.head.next
	return data, nil
}

func (q ListQueue) peek() (data interface{}) {
	if q.head == nil {
		return nil
	}
	data = q.head.data
	return data
}
