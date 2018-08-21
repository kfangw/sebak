package trie

import (
	"bytes"
	"testing"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/storage"
)

func TestEthTrieHash(t *testing.T) {
	var root = sebakcommon.Hash{}
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	defer st.Close()

	db := NewEthDatabase(st)

	trie := NewTrie(root, db)

	hash1 := trie.Hash()

	if err := trie.TryUpdate([]byte("a"), []byte("b")); err != nil {
		t.Fatal(err)
	}

	hash2 := trie.Hash()

	hash3, err := trie.Commit(nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("root:%v hash1:%v hash2:%v hash3:%v", root, hash1, hash2, hash3)

	if bytes.Equal(hash1.Bytes(), hash2.Bytes()) {
		t.Errorf("hash1 (before update) should be equal hash2 (after update)")
	}

	if !bytes.Equal(hash2.Bytes(), hash3.Bytes()) {
		t.Errorf("hash2 should be equal hash3")
	}
}
