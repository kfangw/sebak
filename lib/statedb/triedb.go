package statedb

import (
	"fmt"

	sebak "boscoin.io/sebak/lib"
	common "boscoin.io/sebak/lib/common"
	cstorage "boscoin.io/sebak/lib/contract/storage"
	"boscoin.io/sebak/lib/trie"
)

type StateTrieDB struct {
	db   *trie.EthDatabase
	trie *trie.Trie

	accounts map[string]*sebak.BlockAccount
	storages map[string]*StorageTrieDB
}

func NewStateTrieDB(root common.Hash, db *trie.EthDatabase) *StateTrieDB {
	stdb := &StateTrieDB{
		db:       db,
		trie:     trie.NewTrie(root, db),
		accounts: make(map[string]*sebak.BlockAccount),
		storages: make(map[string]*StorageTrieDB),
	}

	return stdb
}

func (stdb *StateTrieDB) GetAccount(addr string) (*sebak.BlockAccount, error) {
	if a, ok := stdb.accounts[addr]; ok {
		return a, nil

	}
	a, err := sebak.GetBlockAccount(stdb.db.LevelDBBackend(), addr)
	if err != nil {
		return nil, err
	}
	return a, nil

}

/*
func (db *StateTrieDB) SetDeployCode(addr string,deployCode *payload.DeployCode) error {

}
*/

func (db *StateTrieDB) PutStorageItem(addr, key string, item *cstorage.StorageItem) error {
	if s, ok := db.storages[addr]; ok {
		s.PutStorageItem(key, item)
		return nil
	} else {
		a, err := db.GetAccount(addr)
		if err != nil {
			return err
		}

		storage := NewStorageTrieDB(addr, a.StorageRoot, db.db)
		storage.PutStorageItem(key, item)
		db.storages[addr] = storage
	}

	return nil
}

func (db *StateTrieDB) updateStorageRoot(addr string, storageRoot common.Hash) error {
	acc, err := db.GetAccount(addr)
	if err != nil {
		return err
	}

	acc.StorageRoot = storageRoot
	db.accounts[addr] = acc

	return db.updateTrie(addr, acc)
}

func (db *StateTrieDB) updateCodeHash(addr string, hash common.Hash) error {
	acc, err := db.GetAccount(addr)
	if err != nil {
		return err
	}
	acc.CodeHash = hash
	db.accounts[addr] = acc
	//TODO:
	return db.updateTrie(addr, acc)
}

func (db *StateTrieDB) updateTrie(addr string, account *sebak.BlockAccount) error {
	addrHash, err := trie.EncodeToBytes([]byte(addr))
	if err != nil {
		return err
	}
	accountHash, err := trie.EncodeToBytes(account)
	if err != nil {
		return err
	}

	if err := db.trie.TryUpdate(addrHash, accountHash); err != nil {
		return err
	}
	return nil
}

func (db *StateTrieDB) Hash() common.Hash {
	return db.trie.Hash()
}

func (db *StateTrieDB) CommitTrie() (root common.Hash, err error) {

	for addr, sdb := range db.storages {
		sRoot, err := sdb.CommitTrie()
		if err != nil {
			return common.EmptyHash, err
		}
		if err := db.updateStorageRoot(addr, sRoot); err != nil {
			return common.EmptyHash, err
		}
	}

	return db.trie.Commit(nil)
}

func (stdb *StateTrieDB) CommitDB(root common.Hash) error {
	stdb.trie.CommitDB(root)

	levelDB := stdb.db.LevelDBBackend()

	for addr, stg := range stdb.storages {
		acc, ok := stdb.accounts[addr]
		if !ok {
			return fmt.Errorf("Account doesn't exist of this storage")
		}
		sRoot := acc.StorageRoot
		if err := stg.CommitDB(sRoot); err != nil {
			return err
		}
	}

	for _, acc := range stdb.accounts {
		if err := acc.Save(levelDB); err != nil {
			return err
		}
	}

	return nil
}
