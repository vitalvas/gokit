package arccache

import "time"

// element is a doubly linked list node used in the ARC cache.
type element[K comparable, V any] struct {
	key       K
	value     V
	expiresAt time.Time
	ghost     bool        // true if this is a ghost entry (key only, no value)
	list      *list[K, V] // which list this element belongs to
	prev      *element[K, V]
	next      *element[K, V]
}

// list is a doubly linked list with O(1) push, remove, and move operations.
type list[K comparable, V any] struct {
	root element[K, V] // sentinel
	size int
}

func newList[K comparable, V any]() *list[K, V] {
	l := &list[K, V]{}
	l.root.prev = &l.root
	l.root.next = &l.root
	return l
}

func (l *list[K, V]) len() int {
	return l.size
}

// head returns the first element or nil if empty.
func (l *list[K, V]) head() *element[K, V] {
	if l.size == 0 {
		return nil
	}
	return l.root.next
}

// tail returns the last element or nil if empty.
func (l *list[K, V]) tail() *element[K, V] {
	if l.size == 0 {
		return nil
	}
	return l.root.prev
}

// pushFront inserts e at the front of the list.
func (l *list[K, V]) pushFront(e *element[K, V]) {
	e.prev = &l.root
	e.next = l.root.next
	l.root.next.prev = e
	l.root.next = e
	l.size++
}

// remove removes e from the list.
func (l *list[K, V]) remove(e *element[K, V]) {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.prev = nil
	e.next = nil
	l.size--
}

// moveToFront moves e to the front of the list.
func (l *list[K, V]) moveToFront(e *element[K, V]) {
	if l.root.next == e {
		return
	}
	e.prev.next = e.next
	e.next.prev = e.prev
	e.prev = &l.root
	e.next = l.root.next
	l.root.next.prev = e
	l.root.next = e
}

// clear removes all elements from the list.
func (l *list[K, V]) clear() {
	l.root.prev = &l.root
	l.root.next = &l.root
	l.size = 0
}
