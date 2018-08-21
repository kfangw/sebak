package statedb

import (
	"testing"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/contract/storage"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/trie"
)

func TestStateDBPutStorageItem(t *testing.T) {
	var root = sebakcommon.Hash{}
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	defer st.Close()

	db := trie.NewEthDatabase(st)
	sdb := NewStateTrieDB(root, db)

	var (
		addr       = "helloworld"
		fund       = sebakcommon.Amount(1000)
		checkpoint = "tx01-tx01"
		itemkey    = "key1"
	)
	if err := sdb.CreateAccount(addr, fund, checkpoint); err != nil {
		t.Fatal(err)
	}

	item := storage.NewStorageItem(addr, itemkey)
	if err := sdb.PutStorageItem(addr, itemkey, item); err != nil {
		t.Fatal(err)
	}

	newRoot, err := sdb.CommitTrie()
	if err != nil {
		t.Fatal(err)
	}

	if root == newRoot {
		t.Fatalf("root and newRoot root:%v newRoot:%v", root, newRoot)
	}

	if err := sdb.CommitDB(newRoot); err != nil {
		t.Fatal(err)
	}

	sdb2 := NewStateTrieDB(newRoot, db)

	item2, err := sdb2.GetStorageItem(addr, itemkey)
	if err != nil {
		t.Fatal(err)
	}

	if item2 == nil {
		t.Error("item2 is nil")
	}
}

func TestStateDBCreateAccount(t *testing.T) {
	var root = sebakcommon.Hash{}
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	defer st.Close()

	db := trie.NewEthDatabase(st)
	sdb := NewStateTrieDB(root, db)

	var (
		addr       = "helloworld"
		fund       = sebakcommon.Amount(1000)
		checkpoint = "tx01-tx01"
	)

	if err := sdb.CreateAccount(addr, fund, checkpoint); err != nil {
		t.Fatal(err)
	}

	root2, err := sdb.CommitTrie()
	if err != nil {
		t.Fatal(err)
	}

	if root == root2 {
		t.Fatalf("root and root2 are equal  root:%v root2:%v", root, root2)
	}

	if err := sdb.CommitDB(root2); err != nil {
		t.Fatal(err)
	}

	sdb2 := NewStateTrieDB(root2, db)

	fund1, err := sdb2.GetBalance(addr)
	if err != nil {
		t.Fatal(err)
	}
	if fund1 != fund {
		t.Fatalf("fund != fund1 have:%v want:%v", fund1, fund)
	}
}
