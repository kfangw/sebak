package statedb

import (
	common "boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/contract/storage"
	"boscoin.io/sebak/lib/trie"
)

type StorageTrieDB struct {
	addr  string // account's address
	db    *trie.EthDatabase
	trie  *trie.Trie
	items map[string]*storage.StorageItem
}

func NewStorageTrieDB(addr string, root common.Hash, db *trie.EthDatabase) *StorageTrieDB {
	stdb := &StorageTrieDB{
		addr:  addr,
		db:    db,
		trie:  trie.NewTrie(root, db),
		items: make(map[string]*storage.StorageItem),
	}
	return stdb
}

func (s *StorageTrieDB) GetStorageItem(key string) (*storage.StorageItem, error) {
	if item, ok := s.items[key]; ok {
		return item, nil
	}

	dbBackend := s.db.LevelDBBackend()

	item, err := storage.GetStorageItem(dbBackend, s.addr, key)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s *StorageTrieDB) PutStorageItem(key string, item *storage.StorageItem) error {
	s.items[key] = item
	keyHash, err := trie.EncodeToBytes(key)
	if err != nil {
		return err
	}
	itemHash, err := trie.EncodeToBytes(item)
	if err != nil {
		return err
	}
	return s.trie.TryUpdate(keyHash, itemHash)
}

func (s *StorageTrieDB) CommitTrie() (common.Hash, error) {
	return s.trie.Commit(nil)
}

func (s *StorageTrieDB) Hash() common.Hash {
	return s.trie.Hash()
}

func (s *StorageTrieDB) CommitDB(root common.Hash) error {
	s.trie.CommitDB(root)
	return nil
}
