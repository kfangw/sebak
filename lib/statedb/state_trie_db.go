package statedb

import (
	"fmt"

	"boscoin.io/sebak/lib/block"
	common "boscoin.io/sebak/lib/common"
	cstorage "boscoin.io/sebak/lib/contract/storage"
	"boscoin.io/sebak/lib/trie"
)

type StateTrieDB struct {
	db   *trie.EthDatabase
	trie *trie.Trie

	accounts map[string]*block.BlockAccount
	storages map[string]*StorageTrieDB
}

func NewStateTrieDB(root common.Hash, db *trie.EthDatabase) *StateTrieDB {
	stdb := &StateTrieDB{
		db:       db,
		trie:     trie.NewTrie(root, db),
		accounts: make(map[string]*block.BlockAccount),
		storages: make(map[string]*StorageTrieDB),
	}

	return stdb
}

/*
//TODO:
func (db *StateTrieDB) SetDeployCode(addr string,deployCode *payload.DeployCode) error {

}
*/

func (db *StateTrieDB) CreateAccount(addr string, fund common.Amount, checkpoint string) error {
	account := block.NewBlockAccount(addr, fund, checkpoint)
	db.accounts[addr] = account
	return db.updateTrie(addr, account)
}

func (db *StateTrieDB) GetBalance(addr string) (common.Amount, error) {
	a, err := db.getAccount(addr) // doesn't update db.accounts
	if err != nil {
		return common.Amount(0), err
	}

	amount, err := common.AmountFromString(a.Balance)
	if err != nil {
		return common.Amount(0), err
	}
	return amount, nil
}

func (db *StateTrieDB) DepositBalance(addr string, amount common.Amount) error {
	a, err := db.loadAccount(addr)
	if err != nil {
		return err
	}

	//TODO: checkpoint?
	if err := a.Deposit(amount, "tx1-tx1"); err != nil {
		return err
	}

	return db.updateTrie(addr, a)
}

func (db *StateTrieDB) WithdrawBalance(addr string, amount common.Amount) error {
	a, err := db.loadAccount(addr)
	if err != nil {
		return err
	}

	//TODO: checkpoint?
	if err := a.Withdraw(amount, "tx1-tx1"); err != nil {
		return err
	}
	return db.updateTrie(addr, a)
}

func (db *StateTrieDB) PutStorageItem(addr, key string, item *cstorage.StorageItem) error {
	s, err := db.loadStorageTrieDB(addr)
	if err != nil {
		return err
	}

	if err := s.PutStorageItem(key, item); err != nil {
		return err
	}
	return nil
}

func (db *StateTrieDB) GetStorageItem(addr, key string) (*cstorage.StorageItem, error) {
	s, err := db.loadStorageTrieDB(addr)
	if err != nil {
		return nil, err
	}
	item, err := s.GetStorageItem(key)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (db *StateTrieDB) updateStorageRoot(addr string, storageRoot common.Hash) error {
	acc, err := db.loadAccount(addr)
	if err != nil {
		return err
	}

	acc.StorageRoot = storageRoot
	db.accounts[addr] = acc

	return db.updateTrie(addr, acc)
}

func (db *StateTrieDB) updateCodeHash(addr string, hash common.Hash) error {
	acc, err := db.loadAccount(addr)
	if err != nil {
		return err
	}
	acc.CodeHash = hash
	db.accounts[addr] = acc
	//TODO:
	return db.updateTrie(addr, acc)
}

func (db *StateTrieDB) updateTrie(addr string, account *block.BlockAccount) error {
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

func (stdb *StateTrieDB) getAccount(addr string) (*block.BlockAccount, error) {
	if a, ok := stdb.accounts[addr]; ok {
		return a, nil
	}

	addrB, err := trie.EncodeToBytes(addr)
	if err != nil {
		return nil, err
	}

	value, err := stdb.trie.TryGet(addrB)
	if err != nil {
		return nil, err
	}

	var account block.BlockAccount
	if err := trie.DecodeBytes(value, &account); err != nil {
		return nil, err
	}

	return &account, nil
}

func (stdb *StateTrieDB) loadAccount(addr string) (*block.BlockAccount, error) {
	if a, ok := stdb.accounts[addr]; ok {
		return a, nil

	}
	addrB, err := trie.EncodeToBytes(addr)
	if err != nil {
		return nil, err
	}

	value, err := stdb.trie.TryGet(addrB)
	if err != nil {
		return nil, err
	}

	var account block.BlockAccount
	if err := trie.DecodeBytes(value, &account); err != nil {
		return nil, err
	}

	stdb.accounts[addr] = &account
	return &account, nil
}

func (stdb *StateTrieDB) loadStorageTrieDB(addr string) (*StorageTrieDB, error) {
	if s, ok := stdb.storages[addr]; ok {
		return s, nil
	}

	a, err := stdb.loadAccount(addr)
	if err != nil {
		return nil, err
	}

	s := NewStorageTrieDB(addr, a.StorageRoot, stdb.db)
	stdb.storages[addr] = s
	return s, nil
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
