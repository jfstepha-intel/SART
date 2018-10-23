package queue

type element struct {
	value interface{}
	next  *element
}

func newelement(value interface{}) *element {
	return &element{
		value,
		nil,
	}
}

type Queue struct {
	head   *element
	tail   *element
	length int
}

func New() *Queue {
	return &Queue{}
}

func (q *Queue) Push(value interface{}) {
	e := newelement(value)

	if q.length == 0 {
		q.head = e
	} else {
		q.tail.next = e
	}

	q.tail = e
	q.length++
}

func (q *Queue) Pop() interface{} {
	if q.length == 0 {
		return nil
	}

	e := q.head
	q.head = q.head.next
	q.length--
	return e.value
}

func (q Queue) Len() int {
	return q.length
}

func (q Queue) Values() (values []interface{}) {
	cur := q.head
	for cur != nil {
		values = append(values, cur.value)
		cur = cur.next
	}
	return
}

func (q Queue) Empty() bool {
	return q.length == 0
}
