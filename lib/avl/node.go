package avl

// NOTE: This file favors int64 as opposed to int for size/counts.
// The Tree on the other hand favors int.  This is intentional.

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto/tmhash"
	cmn "github.com/tendermint/tendermint/libs/common"
	"io"
)

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

	leftNode  *Node `rlp:"-"`
	rightNode *Node `rlp:"-"`
	persisted bool `rlp:"-"`
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
// The new node doesn't have its hash saved or set. The caller must set it
// afterwards.

func (node *Node) EncodeRLP(w io.Writer) (err error) {
	var cause error
	cause = rlp.Encode(w, node.height)
	if cause != nil {
		return cmn.ErrorWrap(cause, "writing height")
	}
	cause = rlp.Encode(w, node.size)
	if cause != nil {
		return cmn.ErrorWrap(cause, "writing size")
	}
	cause = rlp.Encode(w, node.version)
	if cause != nil {
		return cmn.ErrorWrap(cause, "writing version")
	}

	// Unlike writeHashBytes, key is written for inner nodes.
	cause = rlp.Encode(w, node.key)
	if cause != nil {
		return cmn.ErrorWrap(cause, "writing key")
	}

	if node.isLeaf() {
		cause = rlp.Encode(w, node.value)
		if cause != nil {
			return cmn.ErrorWrap(cause, "writing value")
		}
	} else {
		if node.leftHash == nil {
			panic("node.leftHash was nil in writeBytes")
		}
		cause = rlp.Encode(w, node.leftHash)
		if cause != nil {
			return cmn.ErrorWrap(cause, "writing left hash")
		}

		if node.rightHash == nil {
			panic("node.rightHash was nil in writeBytes")
		}
		cause = rlp.Encode(w, node.rightHash)
		if cause != nil {
			return cmn.ErrorWrap(cause, "writing right hash")
		}
	}
	return nil
}
func (node *Node) DecodeRLP(s *rlp.Stream) (error) {
	height, cause := s.Uint()
	if cause != nil {
		return cmn.ErrorWrap(cause, "decoding node.height")
	}
	size, cause := s.Uint()
	if cause != nil {
		return cmn.ErrorWrap(cause, "decoding node.size")
	}
	ver, cause := s.Uint()
	if cause != nil {
		return cmn.ErrorWrap(cause, "decoding node.version")
	}
	key, cause := s.Bytes()
	if cause != nil {
		return cmn.ErrorWrap(cause, "decoding node.key")
	}

	node.height = height
	node.size = size
	node.version = ver
	node.key = key

	// Read node body.
	if node.isLeaf() {
		val, cause := s.Bytes()
		if cause != nil {
			return cmn.ErrorWrap(cause, "decoding node.value")
		}
		node.value = val
	} else { // Read children.
		leftHash, cause := s.Bytes()
		if cause != nil {
			return cmn.ErrorWrap(cause, "deocding node.leftHash")
		}

		rightHash, cause := s.Bytes()
		if cause != nil {
			return cmn.ErrorWrap(cause, "decoding node.rightHash")
		}
		node.leftHash = leftHash
		node.rightHash = rightHash
	}
	return nil
}
func MakeNode(buf []byte) (*Node, cmn.Error) {
	var n Node
	cause := rlp.DecodeBytes(buf, &n)
	if cause != nil {
		return nil, cmn.ErrorWrap(cause, "decoding rlp")
	}
	return &n, nil
}

// String returns a string representation of the node.
func (node *Node) String() string {
	hashstr := "<no hash>"
	if len(node.hash) > 0 {
		hashstr = fmt.Sprintf("%X", node.hash)
	}
	return fmt.Sprintf("Node{%s:%s@%d %X;%X}#%s",
		cmn.ColoredBytes(node.key, cmn.Green, cmn.Blue),
		cmn.ColoredBytes(node.value, cmn.Cyan, cmn.Blue),
		node.version,
		node.leftHash, node.rightHash,
		hashstr)
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

func (node *Node) isLeaf() bool {
	return node.height == 0
}
// Computes the hash of the node without computing its descendants. Must be
// called on nodes which have descendant node hashes already computed.
func (node *Node) _hash() []byte {
	if node.hash != nil {
		return node.hash
	}

	h := tmhash.New()
	buf := new(bytes.Buffer)
	if err := node.writeHashBytes(buf); err != nil {
		panic(err)
	}
	h.Write(buf.Bytes())
	node.hash = h.Sum(nil)

	return node.hash
}

// Hash the node and its descendants recursively. This usually mutates all
// descendant nodes. Returns the node hash and number of nodes hashed.
func (node *Node) hashWithCount() ([]byte, int64) {
	if node.hash != nil {
		return node.hash, 0
	}

	h := tmhash.New()
	buf := new(bytes.Buffer)
	hashCount, err := node.writeHashBytesRecursively(buf)
	if err != nil {
		panic(err)
	}
	h.Write(buf.Bytes())
	node.hash = h.Sum(nil)

	return node.hash, hashCount + 1
}

// Writes the node's hash to the given io.Writer. This function expects
// child hashes to be already set.
func (node *Node) writeHashBytes(w io.Writer) cmn.Error {
	err := amino.EncodeInt8(w, node.height)
	if err != nil {
		return cmn.ErrorWrap(err, "writing height")
	}
	err = amino.EncodeVarint(w, node.size)
	if err != nil {
		return cmn.ErrorWrap(err, "writing size")
	}
	err = amino.EncodeVarint(w, node.version)
	if err != nil {
		return cmn.ErrorWrap(err, "writing version")
	}

	// Key is not written for inner nodes, unlike writeBytes.

	if node.isLeaf() {
		err = amino.EncodeByteSlice(w, node.key)
		if err != nil {
			return cmn.ErrorWrap(err, "writing key")
		}
		// Indirection needed to provide proofs without values.
		// (e.g. proofLeafNode.ValueHash)
		valueHash := tmhash.Sum(node.value)
		err = amino.EncodeByteSlice(w, valueHash)
		if err != nil {
			return cmn.ErrorWrap(err, "writing value")
		}
	} else {
		if node.leftHash == nil || node.rightHash == nil {
			panic("Found an empty child hash")
		}
		err = amino.EncodeByteSlice(w, node.leftHash)
		if err != nil {
			return cmn.ErrorWrap(err, "writing left hash")
		}
		err = amino.EncodeByteSlice(w, node.rightHash)
		if err != nil {
			return cmn.ErrorWrap(err, "writing right hash")
		}
	}

	return nil
}

// Writes the node's hash to the given io.Writer.
// This function has the side-effect of calling hashWithCount.
func (node *Node) writeHashBytesRecursively(w io.Writer) (hashCount int64, err cmn.Error) {
	if node.leftNode != nil {
		leftHash, leftCount := node.leftNode.hashWithCount()
		node.leftHash = leftHash
		hashCount += leftCount
	}
	if node.rightNode != nil {
		rightHash, rightCount := node.rightNode.hashWithCount()
		node.rightHash = rightHash
		hashCount += rightCount
	}
	err = node.writeHashBytes(w)

	return
}


// Writes the node as a serialized byte slice to the supplied io.Writer.
func (node *Node) writeBytes(w io.Writer) cmn.Error {
	var cause error
	cause = amino.EncodeInt8(w, node.height)
	if cause != nil {
		return cmn.ErrorWrap(cause, "writing height")
	}
	cause = amino.EncodeVarint(w, node.size)
	if cause != nil {
		return cmn.ErrorWrap(cause, "writing size")
	}
	cause = amino.EncodeVarint(w, node.version)
	if cause != nil {
		return cmn.ErrorWrap(cause, "writing version")
	}

	// Unlike writeHashBytes, key is written for inner nodes.
	cause = amino.EncodeByteSlice(w, node.key)
	if cause != nil {
		return cmn.ErrorWrap(cause, "writing key")
	}

	if node.isLeaf() {
		cause = amino.EncodeByteSlice(w, node.value)
		if cause != nil {
			return cmn.ErrorWrap(cause, "writing value")
		}
	} else {
		if node.leftHash == nil {
			panic("node.leftHash was nil in writeBytes")
		}
		cause = amino.EncodeByteSlice(w, node.leftHash)
		if cause != nil {
			return cmn.ErrorWrap(cause, "writing left hash")
		}

		if node.rightHash == nil {
			panic("node.rightHash was nil in writeBytes")
		}
		cause = amino.EncodeByteSlice(w, node.rightHash)
		if cause != nil {
			return cmn.ErrorWrap(cause, "writing right hash")
		}
	}
	return nil
}

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
func (node *Node) get(ndb *nodeDB, key []byte) (index int64, value []byte) {
	if node.isLeaf() {
		switch bytes.Compare(node.key, key) {
		case -1:
			return 1, nil
		case 1:
			return 0, nil
		default:
			return 0, node.value
		}
	}

	if bytes.Compare(key, node.key) < 0 {
		return node.getLeftNode(ndb).get(ndb, key)
	}
	rightNode := node.getRightNode(ndb)
	index, value = rightNode.get(ndb, key)
	index += node.size - rightNode.size
	return index, value
}

func (node *Node) getByIndex(ndb *nodeDB, index int64) (key []byte, value []byte) {
	if node.isLeaf() {
		if index == 0 {
			return node.key, node.value
		}
		return nil, nil
	}
	// TODO: could improve this by storing the
	// sizes as well as left/right hash.
	leftNode := node.getLeftNode(ndb)

	if index < leftNode.size {
		return leftNode.getByIndex(ndb, index)
	}
	return node.getRightNode(ndb).getByIndex(ndb, index-leftNode.size)
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

// NOTE: mutates height and size
func (node *Node) calcHeightAndSize(ndb *nodeDB) {
	node.height = maxInt8(node.getLeftNode(ndb).height, node.getRightNode(ndb).height) + 1
	node.size = node.getLeftNode(ndb).size + node.getRightNode(ndb).size
}

func (node *Node) calcBalance(ndb *nodeDB) int {
	return int(node.getLeftNode(ndb).height) - int(node.getRightNode(ndb).height)
}

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


// Only used in testing...
func (node *Node) lmd(ndb *nodeDB) *Node {
	if node.isLeaf() {
		return node
	}
	return node.getLeftNode(ndb).lmd(ndb)
}

func maxInt8(a, b int8) int8 {
	if a > b {
		return a
	}
	return b
}
