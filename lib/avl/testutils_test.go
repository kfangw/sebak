package avl

import (
	"boscoin.io/sebak/lib/db"
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/rlp"
	"runtime"
	"testing"

	mrand "math/rand"

	cmn "github.com/tendermint/tendermint/libs/common"
)

// Only used in testing...
func (node *Node) lmd(ndb *nodeDB) *Node {
	if node.isLeaf() {
		return node
	}
	return node.getLeftNode(ndb).lmd(ndb)
}

func randstr(length int) string {
	return cmn.RandStr(length)
}

func i2b(i int) []byte {
	buf := new(bytes.Buffer)
	rlp.Encode(buf, uint64(i))
	return buf.Bytes()
}

func b2i(bz []byte) int {
	var uint64_ uint64
	rlp.DecodeBytes(bz, &uint64_)
	return int(uint64_)
}

// Convenience for a new node
func N(l, r interface{}) *Node {
	var left, right *Node
	if _, ok := l.(*Node); ok {
		left = l.(*Node)
	} else {
		left = NewNode(i2b(l.(int)), nil, 0)
	}
	if _, ok := r.(*Node); ok {
		right = r.(*Node)
	} else {
		right = NewNode(i2b(r.(int)), nil, 0)
	}

	n := &Node{
		key:       right.lmd(nil).key,
		value:     nil,
		leftNode:  left,
		rightNode: right,
	}
	n.calcHeightAndSize(nil)
	return n
}

// Setup a deep node
func T(n *Node) *MutableTree {
	t := NewMutableTree(db.NewMemDB(), 0)

	n.hashRecursively(func(n *Node) {})
	t.root = n
	return t
}

// Convenience for simple printing of keys & tree structure
func P(n *Node) string {
	if n.height == 0 {
		return fmt.Sprintf("%v", b2i(n.key))
	}
	return fmt.Sprintf("(%v %v)", P(n.leftNode), P(n.rightNode))
}

func randBytes(length int) []byte {
	key := make([]byte, length)
	// math.rand.Read always returns err=nil
	mrand.Read(key)
	return key
}

type traverser struct {
	first string
	last  string
	count int
}

func (t *traverser) view(key, value []byte) bool {
	if t.first == "" {
		t.first = string(key)
	}
	t.last = string(key)
	t.count++
	return false
}

func expectTraverse(t *testing.T, trav traverser, start, end string, count int) {
	if trav.first != start {
		t.Error("Bad start", start, trav.first)
	}
	if trav.last != end {
		t.Error("Bad end", end, trav.last)
	}
	if trav.count != count {
		t.Error("Bad count", count, trav.count)
	}
}

func BenchmarkImmutableAvlTreeMemDB(b *testing.B) {
	benchmarkImmutableAvlTreeWithDB(b, db.NewMemDB())
}

func benchmarkImmutableAvlTreeWithDB(b *testing.B, db db.DB) {
	defer db.Close()

	b.StopTimer()

	t := NewMutableTree(db, 100000)
	value := []byte{}
	for i := 0; i < 1000000; i++ {
		t.Set(i2b(int(cmn.RandInt32())), value)
		if i > 990000 && i%1000 == 999 {
			t.SaveVersion()
		}
	}
	b.ReportAllocs()
	t.SaveVersion()

	runtime.GC()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		ri := i2b(int(cmn.RandInt32()))
		t.Set(ri, value)
		t.Remove(ri)
		if i%100 == 99 {
			t.SaveVersion()
		}
	}
}
