package statedb

import (
	common "boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/contract/storage"
)

type StateDBWriter interface {
	//Account states
	CreateAccount(addr string, fund common.Amount, checkpoint string) error
	GetBalance(addr string) (common.Amount, error)
	DepositBalance(addr string, amount common.Amount) error
	WithdrawBalance(addr string, amount common.Amount) error
	//SetDeployCode

	//Account's Storage states
	GetStorageItem(addr string, key string) (*storage.StorageItem, error)
	PutStorageItem(addr string, key string, item *storage.StorageItem) error

	// State management
	Hash() common.Hash
	CommitTrie() (root common.Hash, err error)
	CommitDB(root common.Hash) error
}

type StateDBReader interface {
	//Account states
	GetBalance(addr string) (common.Amount, error)
	// ? GetAccount(addr) (*sebak.BlockAccount,error)

	GetStorageItem(addr string, key string) (*storage.StorageItem, error)
}
