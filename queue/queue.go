package queue

import (
	"fmt"
	"reflect"
)

// ListQueue - It's a queue with linked lists dawg
type ListQueue struct {
	root     *Node
	dataType reflect.Type
	length   int
}

type Node struct {
	next *Node
	prev *Node
	Data interface{}
}

func (n Node) String() string {
	return fmt.Sprintf("%v", n.Data)
}

func (q ListQueue) String() string {
	s := "["
	switch q.length {
	case 0:
		s += "]"
		return s
	case 1:
		s = fmt.Sprintf("%v%v]", s, q.Start())
		return s
	}
	s += q.Start().String()
	for i := q.Start().next; i != q.root; i = i.next {
		s = fmt.Sprintf("%v,%v", s, i.String())
	}
	s += "]"
	return s
}

// End Get the last node in the queue
func (q ListQueue) End() *Node {
	return q.root.prev
}

// Start Get the first node in the queue
func (q ListQueue) Start() *Node {
	return q.root.next
}

// Length Length of the queue
func (q ListQueue) Length() int {
	return q.length
}

func NewListQueue(dataType interface{}) (q *ListQueue) {
	node := new(Node)
	node.next = node
	node.prev = node
	node.Data = dataType
	return &ListQueue{dataType: reflect.TypeOf(dataType), root: node}
}

//Remove Remove value at index of the queue
func (q *ListQueue) Remove(index int) (node *Node) {
	if q.length <= index || index < 0 || q.length <= 0 {
		return nil
	}
	if index == 0 {
		return q.Pop()
	}
	node = q.Start()
	for count := 0; count < index; count++ {
		node = node.next
	}

	node.next.prev = node.prev
	node.prev.next = node.next

	node.next = nil
	node.prev = nil
	q.length--

	return node
}

//Push Push data to top of the queue
func (q *ListQueue) Push(data ...interface{}) *Node {
	if len(data) == 0 {
		return nil
	}

	for _, v := range data {
		if reflect.TypeOf(v) != q.dataType {
			continue
		}
		node := new(Node)
		node.Data = v
		node.prev = q.root.prev
		node.next = q.root
		node.next.prev = node
		node.prev.next = node

		q.length++
	}
	return q.End()
}

//Pop Take the top value from bottom of the queue
func (q *ListQueue) Pop() (node *Node) {
	if q.Start() == q.root {
		return nil
	}
	node = q.Start()
	node.next.prev = node.prev
	node.prev.next = node.next
	q.length--
	node.next = nil
	node.prev = nil
	return node
}

//Peek Look at value on bottom of the queue
func (q ListQueue) Peek() (node *Node) {
	if q.Start() == q.root {
		return nil
	}
	return q.Start()
}
