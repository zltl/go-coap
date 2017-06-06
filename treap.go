package coap

type Treap struct {
	compareKey      Compare
	comparePriority Compare
	root            *node
}

// Compare returns an integer comparing the two items
// lexicographically. The result will be 0 if a==b, -1 if a < b, and
// +1 if a > b.
type Compare func(a, b interface{}) int

// TreapItem can be anything.
type TreapItem interface{}

type node struct {
	item  TreapItem
	left  *node
	right *node
}

func NewTreap(ck, cp Compare) *Treap {
	if ck == nil || cp == nil {
		panic("Nil pointer")
	}
	return &Treap{compareKey: ck, comparePriority: cp, root: nil}
}

func (t *Treap) Min() TreapItem {
	n := t.root
	if n == nil {
		return nil
	}
	for n.left != nil {
		n = n.left
	}
	return n.item
}

func (t *Treap) Max() TreapItem {
	n := t.root
	if n == nil {
		return nil
	}
	for n.right != nil {
		n = n.right
	}
	return n.item
}

func (t *Treap) Top() TreapItem {
	return t.root
}

func (t *Treap) Pop() TreapItem {
	n := t.root
	t = t.Delete(n)
	return n
}

func (t *Treap) Get(target TreapItem) TreapItem {
	n := t.root
	for n != nil {
		c := t.compareKey(target, n.item)
		if c < 0 {
			n = n.left
		} else if c > 0 {
			n = n.right
		} else {
			return n.item
		}
	}
	return nil
}

// Note: only the priority of the first insert of an item is used.
// Priorities from future updates on already existing items are
// ignored.  To change the priority for an item, you need to do a
// Delete then an Upsert.
func (t *Treap) Upsert(item TreapItem) *Treap {
	r := t.union(t.root, &node{item: item})
	return &Treap{compareKey: t.compareKey, comparePriority: t.comparePriority, root: r}
}

func (t *Treap) union(this *node, that *node) *node {
	if this == nil {
		return that
	}
	if that == nil {
		return this
	}

	// this > that ?
	if t.comparePriority(this.item, that.item) > 0 {
		left, middle, right := t.split(that, this.item)
		if middle == nil {
			return &node{
				item:  this.item,
				left:  t.union(this.left, left),
				right: t.union(this.right, right),
			}
		}
		return &node{
			item:  middle.item,
			left:  t.union(this.left, left),
			right: t.union(this.right, right),
		}
	}
	// We don't use middle because the "that" has precendence.
	left, _, right := t.split(this, that.item)
	return &node{
		item:  that.item,
		left:  t.union(left, that.left),
		right: t.union(right, that.right),
	}
}

// Splits a treap into two treaps based on a split item "s".
// The result tuple-3 means (left, X, right), where X is either...
// nil - meaning the item s was not in the original treap.
// non-nil - returning the node that had item s.
// The tuple-3's left result treap has items < s,
// and the tuple-3's right result treap has items > s.
func (t *Treap) split(n *node, s TreapItem) (*node, *node, *node) {
	if n == nil {
		return nil, nil, nil
	}
	c := t.compareKey(s, n.item)
	if c == 0 {
		return n.left, n, n.right
	}
	if c < 0 {
		left, middle, right := t.split(n.left, s)
		return left, middle, &node{
			item:  n.item,
			left:  right,
			right: n.right,
		}
	}
	left, middle, right := t.split(n.right, s)
	return &node{
		item:  n.item,
		left:  n.left,
		right: left,
	}, middle, right
}

func (t *Treap) Delete(target TreapItem) *Treap {
	left, _, right := t.split(t.root, target)
	return &Treap{compareKey: t.compareKey, comparePriority: t.comparePriority, root: t.join(left, right)}
}

// All the items from this are < items from that.
func (t *Treap) join(this *node, that *node) *node {
	if this == nil {
		return that
	}
	if that == nil {
		return this
	}
	// this > that ?
	if t.comparePriority(this.item, that.item) > 0 {
		return &node{
			item:  this.item,
			left:  this.left,
			right: t.join(this.right, that),
		}
	}
	return &node{
		item:  that.item,
		left:  t.join(this, that.left),
		right: that.right,
	}
}

type TreapItemVisitor func(i TreapItem) bool

// Visit items greater-than-or-equal to the pivot.
func (t *Treap) VisitAscend(pivot TreapItem, visitor TreapItemVisitor) {
	t.visitAscend(t.root, pivot, visitor)
}

func (t *Treap) visitAscend(n *node, pivot TreapItem, visitor TreapItemVisitor) bool {
	if n == nil {
		return true
	}
	if t.compareKey(pivot, n.item) <= 0 {
		if !t.visitAscend(n.left, pivot, visitor) {
			return false
		}
		if !visitor(n.item) {
			return false
		}
	}
	return t.visitAscend(n.right, pivot, visitor)
}
