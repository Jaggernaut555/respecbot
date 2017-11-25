package queue

import "testing"

func TestQueue(t *testing.T) {
	var sq *SliceQueue
	var lq *ListQueue
	var serr error
	var lerr error

	data1 := []interface{}{1, "lol"}
	data2 := []interface{}{1, 2, 3}

	data3String := []string{"a", "b", "c"}
	for i := 0; i < 10; i++ {
		data3String = append(data3String, data3String...)
	}
	data3 := []interface{}{}
	for _, v := range data3String {
		data3 = append(data3, v)
	}

	sq, serr = newSliceQueue()
	lq, lerr = newListQueue()
	if sq != nil || serr == nil || lq != nil || lerr == nil {
		t.Errorf("Could create queue with no data")
	}

	sq, serr = newSliceQueue(data1...)
	lq, lerr = newListQueue(data1...)
	if sq != nil || serr == nil {
		t.Errorf("Could create queue with mismatched types")
	}
	if lq != nil || lerr == nil {
		t.Errorf("Could create queue with mismatched types")
	}

	sq, serr = newSliceQueue(data2...)
	lq, lerr = newListQueue(data2...)
	if sq == nil || serr != nil {
		t.Error(serr)
	}
	if lq == nil || lerr != nil {
		t.Error(lerr)
	}

	_, serr = sq.pop()
	_, lerr = lq.pop()
	if serr != nil {
		t.Error(serr)
	}
	if lerr != nil {
		t.Error(lerr)
	}

	sq, serr = newSliceQueue(data3...)
	lq, lerr = newListQueue(data3...)
	if sq == nil || serr != nil {
		t.Error(serr)
	}
	if lq == nil || lerr != nil {
		t.Error(lerr)
	}
}

func BenchmarkSliceQueue(b *testing.B) {
	setup := []interface{}{"a", "b", "c", "d", "e"}
	data := []interface{}{}

	for i := 0; i < 1000000; i++ {
		data = append(data, setup)
	}

	q, err := newSliceQueue(data...)
	if err != nil {
		b.Error(err)
	}

	for i := 0; i < len(data); i++ {
		q.peek()
	}

	for v, err := q.pop(); v != nil; v, err = q.pop() {
		if err != nil {
			b.Fatal(err)
		}
	}

	for _, v := range data {
		err = q.push(v)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkListQueue(b *testing.B) {
	setup := []interface{}{"a", "b", "c", "d", "e"}
	data := []interface{}{}

	for i := 0; i < 1000000; i++ {
		data = append(data, setup)
	}

	for _, v := range data {
		data = append(data, v)
	}

	q, err := newListQueue(data...)
	if err != nil {
		b.Error(err)
	}

	for i := 0; i < len(data); i++ {
		q.peek()
	}

	for v, err := q.pop(); v != nil; v, err = q.pop() {
		if err != nil {
			b.Fatal(err)
		}
	}

	for _, v := range data {
		err = q.push(v)
		if err != nil {
			b.Fatal(err)
		}
	}
}
