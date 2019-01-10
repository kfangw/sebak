package avl

import (
	"boscoin.io/sebak/lib/db"
	"fmt"
	"strings"
)

// ImmutableTree is a container for an immutable AVL+ ImmutableTree. Changes are performed by
// swapping the internal root with a new one, while the container is mutable.
// Note that this tree is not thread-safe.
type ImmutableTree struct {
	root    *Node
	ndb     *nodeDB
	version uint64
}

// NewImmutableTree creates both in-memory and persistent instances
func NewImmutableTree(db db.DB, cacheSize int) *ImmutableTree {
	if db == nil {
		// In-memory Tree.
		return &ImmutableTree{}
	}
	return &ImmutableTree{
		// NodeDB-backed Tree.
		ndb: newNodeDB(db, cacheSize),
	}
}

// String returns a string representation of Tree.
func (t *ImmutableTree) String() string {
	leaves := []string{}
	t.Iterate(func(key []byte, val []byte) (stop bool) {
		leaves = append(leaves, fmt.Sprintf("%x: %x", key, val))
		return false
	})
	return "Tree{" + strings.Join(leaves, ", ") + "}"
}

// Size returns the number of leaf nodes in the tree.
func (t *ImmutableTree) Size() uint64 {
	if t.root == nil {
		return 0
	}
	return t.root.size
}

// Version returns the version of the tree.
func (t *ImmutableTree) Version() uint64 {
	return t.version
}

// Height returns the height of the tree.
func (t *ImmutableTree) Height() uint64 {
	if t.root == nil {
		return 0
	}
	return t.root.height
}

// Has returns whether or not a key exists.
func (t *ImmutableTree) Has(key []byte) bool {
	if t.root == nil {
		return false
	}
	return t.root.has(t.ndb, key)
}

// Hash returns the root hash.
func (t *ImmutableTree) Hash() []byte {
	if t.root == nil {
		return nil
	}
	hash := t.root.hashRecursively(func(*Node) {})
	return hash
}

// Get returns the index and value of the specified key if it exists, or nil
// and the next index, if it doesn't.
func (t *ImmutableTree) Get(key []byte) (value []byte) {
	if t.root == nil {
		return nil
	}
	return t.root.get(t.ndb, key)
}

// Iterate iterates over all keys of the tree, in order.
func (t *ImmutableTree) Iterate(fn func(key []byte, value []byte) bool) (stopped bool) {
	if t.root == nil {
		return false
	}
	return t.root.traverse(t.ndb, true, func(node *Node) bool {
		if node.height == 0 {
			return fn(node.key, node.value)
		}
		return false
	})
}

// IterateRange makes a callback for all nodes with key between start and end non-inclusive.
// If either are nil, then it is open on that side (nil, nil is the same as Iterate)
func (t *ImmutableTree) IterateRange(start, end []byte, ascending bool, fn func(key []byte, value []byte) bool) (stopped bool) {
	if t.root == nil {
		return false
	}
	return t.root.traverseInRange(t.ndb, start, end, ascending, false, 0, func(node *Node, _ uint8) bool {
		if node.height == 0 {
			return fn(node.key, node.value)
		}
		return false
	})
}

// IterateRangeInclusive makes a callback for all nodes with key between start and end inclusive.
// If either are nil, then it is open on that side (nil, nil is the same as Iterate)
func (t *ImmutableTree) IterateRangeInclusive(start, end []byte, ascending bool, fn func(key, value []byte, version uint64) bool) (stopped bool) {
	if t.root == nil {
		return false
	}
	return t.root.traverseInRange(t.ndb, start, end, ascending, true, 0, func(node *Node, _ uint8) bool {
		if node.height == 0 {
			return fn(node.key, node.value, node.version)
		}
		return false
	})
}

// Clone creates a clone of the tree.
// Used internally by MutableTree.
func (t *ImmutableTree) clone() *ImmutableTree {
	return &ImmutableTree{
		root:    t.root,
		ndb:     t.ndb,
		version: t.version,
	}
}

// nodeSize is like Size, but includes inner nodes too.
func (t *ImmutableTree) nodeSize() int {
	size := 0
	t.root.traverse(t.ndb, true, func(n *Node) bool {
		size++
		return false
	})
	return size
}
