package queue

import (
	"testing"
)

func TestQueue(t *testing.T) {
	var q *ListQueue
	var node *Node

	data1 := []interface{}{1, "lol", 1.0}
	data2 := []interface{}{1, 2, 3}
	data3String := []string{"a", "b", "c", "d", "e"}
	data3 := []interface{}{}
	for _, v := range data3String {
		data3 = append(data3, v)
	}

	q = NewListQueue(1)
	node = q.Push()
	if node != nil {
		t.Error("Push nothing failed")
	}

	q = NewListQueue(data1[0])
	if q.Start() != q.End() {
		t.Error("Start/End don't work")
	}
	item := q.Peek()
	if item != nil {
		t.Error("Could peek at nothing")
	}
	if q.String() != "[]" {
		t.Error("Queue formed incorrectly")
	}
	q.Push(data1...)
	if q.Length() > 1 {
		t.Error("Could push mismatched types")
	}
	if q.String() != "[1]" {
		t.Error("Queue formed incorrectly")
	}

	q = NewListQueue(5)
	if item = q.Remove(0); item != nil {
		t.Error("Removed invalid index")
	}
	for _, v := range data2 {
		q.Push(v)
	}
	if q.Length() == 0 {
		t.Error("Push failed")
	}
	if q.Start() == q.End() {
		t.Error("Start/End don't work")
	}
	item = q.Peek()
	if item == nil {
		t.Error("Cannot peek")
	}
	item = q.Remove(2)
	if item == nil || item.Data.(int) != 3 {
		t.Error("Remove failed")
	}
	q.Pop()
	item = q.Remove(0)
	if item == nil {
		t.Error("Could not pop")
	}
	item = q.Pop()
	item = q.Pop()
	if item != nil {
		t.Error("Could pop nothing")
	}

	q = NewListQueue(data3[0])
	q.Push(data3...)
	if q.String() != "[a,b,c,d,e]" {
		t.Error("String not printed correctly")
	}
}

func BenchmarkListQueue(b *testing.B) {
	setup := []interface{}{"a", "b", "c", "d", "e"}
	data := []interface{}{}

	for i := 0; i < 1000000; i++ {
		data = append(data, setup)
	}

	q := NewListQueue(data[0])

	for _, v := range data {
		data = append(data, v)
		q.Push(setup...)
	}

	for i := 0; i < len(data); i++ {
		q.Peek()
	}

	for v := q.Pop(); v != nil; v = q.Pop() {
	}

	q.Push(data...)
}
