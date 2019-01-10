package avl

// NOTE: This file favors int64 as opposed to int for size/counts.
// The Tree on the other hand favors int.  This is intentional.

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/rlp"
	"io"
)

// =================================================
// ================ Initializer ====================
// =================================================
// Node represents a node in a Tree.
type Node struct {
	key       []byte
	value     []byte
	version   uint64
	height    uint64
	size      uint64
	hash      []byte
	leftHash  []byte
	rightHash []byte

	leftNode  *Node
	rightNode *Node
	persisted bool
}

// NewNode returns a new node from a key, value and version.
func NewNode(key []byte, value []byte, version uint64) *Node {
	return &Node{
		key:     key,
		value:   value,
		height:  0,
		size:    1,
		version: version,
	}
}

// MakeNode constructs an *Node from an encoded byte slice.
//
func MakeNode(buf []byte) (*Node, error) {
	var n Node
	cause := rlp.DecodeBytes(buf, &n)
	if cause != nil {
		return nil, errors.Wrap(cause, "decoding rlp")
	}
	return &n, nil
}

// clone creates a shallow copy of a node with its hash set to nil.
func (node *Node) clone(version uint64) *Node {
	if node.isLeaf() {
		panic("Attempt to copy a leaf node")
	}
	return &Node{
		key:       node.key,
		height:    node.height,
		version:   version,
		size:      node.size,
		hash:      nil,
		leftHash:  node.leftHash,
		leftNode:  node.leftNode,
		rightHash: node.rightHash,
		rightNode: node.rightNode,
		persisted: false,
	}
}

// =================================================
// ================= Calculate =====================
// =================================================
// NOTE: mutates height and size
func (node *Node) calcHeightAndSize(ndb *nodeDB) {
	leftNodeHeight := node.getLeftNode(ndb).height
	rightNodeHeight := node.getRightNode(ndb).height
	if leftNodeHeight > rightNodeHeight {
		node.height = leftNodeHeight
	} else {
		node.height = rightNodeHeight
	}
	node.height++
	node.size = node.getLeftNode(ndb).size + node.getRightNode(ndb).size
}

func (node *Node) calcBalance(ndb *nodeDB) int {
	return int(node.getLeftNode(ndb).height) - int(node.getRightNode(ndb).height)
}

// String returns a string representation of the node.
func (node *Node) String() string {
	hashstr := "<no hash>"
	if len(node.hash) > 0 {
		hashstr = fmt.Sprintf("%X", node.hash)
	}
	return fmt.Sprintf("Node{@%d %X;%X}#%s",
		node.version,
		node.leftHash, node.rightHash,
		hashstr)
}

func (node *Node) isLeaf() bool {
	return node.height == 0
}

// =================================================
// ================== Enc/Dec ======================
// =================================================
// The new node doesn't have its hash saved or set. The caller must set it
// afterwards.
func (node *Node) EncodeRLP(w io.Writer) (err error) {
	var cause error
	cause = rlp.Encode(w, node.height)
	if cause != nil {
		return errors.Wrap(cause, "writing height")
	}
	cause = rlp.Encode(w, node.size)
	if cause != nil {
		return errors.Wrap(cause, "writing size")
	}
	cause = rlp.Encode(w, node.version)
	if cause != nil {
		return errors.Wrap(cause, "writing version")
	}

	// Unlike writeHashBytes, key is written for inner nodes.
	cause = rlp.Encode(w, node.key)
	if cause != nil {
		return errors.Wrap(cause, "writing key")
	}

	if node.isLeaf() {
		cause = rlp.Encode(w, node.value)
		if cause != nil {
			return errors.Wrap(cause, "writing value")
		}
	} else {
		if node.leftHash == nil {
			panic("node.leftHash was nil in writeBytes")
		}
		cause = rlp.Encode(w, node.leftHash)
		if cause != nil {
			return errors.Wrap(cause, "writing left hash")
		}

		if node.rightHash == nil {
			panic("node.rightHash was nil in writeBytes")
		}
		cause = rlp.Encode(w, node.rightHash)
		if cause != nil {
			return errors.Wrap(cause, "writing right hash")
		}
	}
	return nil
}
func (node *Node) DecodeRLP(s *rlp.Stream) error {
	height, cause := s.Uint()
	if cause != nil {
		return errors.Wrap(cause, "decoding node.height")
	}
	size, cause := s.Uint()
	if cause != nil {
		return errors.Wrap(cause, "decoding node.size")
	}
	ver, cause := s.Uint()
	if cause != nil {
		return errors.Wrap(cause, "decoding node.version")
	}
	key, cause := s.Bytes()
	if cause != nil {
		return errors.Wrap(cause, "decoding node.key")
	}

	node.height = height
	node.size = size
	node.version = ver
	node.key = key

	// Read node body.
	if node.isLeaf() {
		val, cause := s.Bytes()
		if cause != nil {
			return errors.Wrap(cause, "decoding node.value")
		}
		node.value = val
	} else { // Read children.
		leftHash, cause := s.Bytes()
		if cause != nil {
			return errors.Wrap(cause, "deocding node.leftHash")
		}

		rightHash, cause := s.Bytes()
		if cause != nil {
			return errors.Wrap(cause, "decoding node.rightHash")
		}
		node.leftHash = leftHash
		node.rightHash = rightHash
	}
	return nil
}

// =================================================
// ================== Hashing ======================
// =================================================

// Hash the node and its descendants recursively. This usually mutates all
// descendant nodes. Returns the node hash and number of nodes hashed.
func (node *Node) hashRecursively(cb func(n *Node)) []byte {
	if node.hash != nil {
		return node.hash
	}

	buf := new(bytes.Buffer)
	if node.leftNode != nil {
		node.leftHash = node.leftNode.hashRecursively(cb)
	}
	if node.rightNode != nil {
		node.rightHash = node.rightNode.hashRecursively(cb)
	}
	err := node.EncodeRLP(buf)
	if err != nil {
		panic(err)
	}
	node.hash = common.MakeHash(buf.Bytes())

	cb(node)

	return node.hash
}

// =================================================
// =================== Getters =====================
// =================================================
// Check if the node has a descendant with the given key.
func (node *Node) has(ndb *nodeDB, key []byte) (has bool) {
	if bytes.Equal(node.key, key) {
		return true
	}
	if node.isLeaf() {
		return false
	}
	if bytes.Compare(key, node.key) < 0 {
		return node.getLeftNode(ndb).has(ndb, key)
	}
	return node.getRightNode(ndb).has(ndb, key)
}

// Get a key under the node.
func (node *Node) get(ndb *nodeDB, key []byte) (value []byte) {
	if node.isLeaf() {
		if bytes.Compare(node.key, key) == 0 {
			return node.value
		} else {
			return nil
		}
	}

	if bytes.Compare(key, node.key) < 0 {
		return node.getLeftNode(ndb).get(ndb, key)
	}
	return node.getRightNode(ndb).get(ndb, key)
}

func (node *Node) getLeftNode(ndb *nodeDB) *Node {
	if node.leftNode != nil {
		return node.leftNode
	}
	return ndb.GetNode(node.leftHash)
}

func (node *Node) getRightNode(ndb *nodeDB) *Node {
	if node.rightNode != nil {
		return node.rightNode
	}
	return ndb.GetNode(node.rightHash)
}

// =================================================
// ================== Traverse =====================
// =================================================

// traverse is a wrapper over traverseInRange when we want the whole tree
func (node *Node) traverse(ndb *nodeDB, ascending bool, cb func(*Node) bool) bool {
	return node.traverseInRange(ndb, nil, nil, ascending, false, 0, func(node *Node, depth uint8) bool {
		return cb(node)
	})
}

func (node *Node) traverseWithDepth(ndb *nodeDB, ascending bool, cb func(*Node, uint8) bool) bool {
	return node.traverseInRange(ndb, nil, nil, ascending, false, 0, cb)
}

func (node *Node) traverseInRange(ndb *nodeDB, start, end []byte, ascending bool, inclusive bool, depth uint8, cb func(*Node, uint8) bool) bool {
	afterStart := start == nil || bytes.Compare(start, node.key) < 0
	startOrAfter := start == nil || bytes.Compare(start, node.key) <= 0
	beforeEnd := end == nil || bytes.Compare(node.key, end) < 0
	if inclusive {
		beforeEnd = end == nil || bytes.Compare(node.key, end) <= 0
	}

	// Run callback per inner/leaf node.
	stop := false
	if !node.isLeaf() || (startOrAfter && beforeEnd) {
		stop = cb(node, depth)
		if stop {
			return stop
		}
	}
	if node.isLeaf() {
		return stop
	}

	if ascending {
		// check lower nodes, then higher
		if afterStart {
			stop = node.getLeftNode(ndb).traverseInRange(ndb, start, end, ascending, inclusive, depth+1, cb)
		}
		if stop {
			return stop
		}
		if beforeEnd {
			stop = node.getRightNode(ndb).traverseInRange(ndb, start, end, ascending, inclusive, depth+1, cb)
		}
	} else {
		// check the higher nodes first
		if beforeEnd {
			stop = node.getRightNode(ndb).traverseInRange(ndb, start, end, ascending, inclusive, depth+1, cb)
		}
		if stop {
			return stop
		}
		if afterStart {
			stop = node.getLeftNode(ndb).traverseInRange(ndb, start, end, ascending, inclusive, depth+1, cb)
		}
	}

	return stop
}
