package block

import (
	"fmt"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
)

// BlockOperation is `Operation` data for block. the storage should support,
//  * find by `Hash`
//  * find by `TxHash`
//
//  * get list by `Source` and created order
//  * get list by `Target` and created order

type BlockOperation struct {
	Hash string `json:"hash"`

	OpHash string `json:"op_hash"`
	TxHash string `json:"tx_hash"`

	Type    operation.OperationType `json:"type"`
	Source  string                  `json:"source"`
	Target  string                  `json:"target"`
	Body    []byte                  `json:"body"`
	Height  uint64                  `json:"block_height"`
	Index   uint64                  `json:"index"`
	TxIndex uint64                  `json:"tx_index"`

	// bellows will be used only for `Save` time.
	transaction transaction.Transaction
	operation   operation.Operation
	linked      string
	txIndex     uint64
	isSaved     bool
	order       *BlockOrder
}

func NewBlockOperationKey(opHash, txHash string, index uint64) string {
	return fmt.Sprintf("%s-%s-%d", opHash, txHash, index)
}

func NewBlockOperationFromOperation(op operation.Operation, tx transaction.Transaction, blockHeight, txIndex, index uint64) (BlockOperation, error) {
	body, err := op.B.Serialize()
	if err != nil {
		return BlockOperation{}, err
	}

	opHash := op.MakeHashString()
	txHash := tx.GetHash()

	target := ""
	if pop, ok := op.B.(operation.Targetable); ok {
		target = pop.TargetAddress()
	}

	linked := ""
	if createAccount, ok := op.B.(operation.CreateAccount); ok {
		if createAccount.Linked != "" {
			linked = createAccount.Linked
		}
	}
	order := NewBlockOpOrder(blockHeight, txIndex, index)

	return BlockOperation{
		Hash: NewBlockOperationKey(opHash, txHash, index),

		OpHash: opHash,
		TxHash: txHash,

		Type:    op.H.Type,
		Source:  tx.B.Source,
		Target:  target,
		Body:    body,
		Height:  blockHeight,
		Index:   index,
		TxIndex: txIndex,

		transaction: tx,
		operation:   op,
		linked:      linked,
		txIndex:     txIndex,
		order:       order,
	}, nil
}

func (bo *BlockOperation) hasTarget() bool {
	if bo.Target != "" {
		return true
	}
	return false
}

func (bo *BlockOperation) targetIsLinked() bool {
	if bo.hasTarget() && bo.linked != "" {
		return true
	}
	return false
}

func (bo *BlockOperation) Save(st *storage.LevelDBBackend) (err error) {
	if bo.isSaved {
		return errors.AlreadySaved
	}

	key := GetBlockOperationKey(bo.Hash)

	var exists bool
	if exists, err = st.Has(key); err != nil {
		return
	} else if exists {
		return errors.BlockAlreadyExists
	}

	if err = st.New(key, bo); err != nil {
		return
	}
	if err = st.New(bo.NewBlockOperationTxHashKey(), bo.Hash); err != nil {
		return
	}

	if err = st.New(bo.NewBlockOperationSourceKey(), bo.Hash); err != nil {
		return
	}
	if err = st.New(bo.NewBlockOperationSourceAndTypeKey(), bo.Hash); err != nil {
		return
	}
	if err = st.New(bo.NewBlockOperationPeersKey(bo.Source), bo.Hash); err != nil {
		return
	}
	if err = st.New(bo.NewBlockOperationPeersAndTypeKey(bo.Source), bo.Hash); err != nil {
		return
	}
	if err = st.New(bo.NewBlockOperationBlockHeightKey(), bo.Hash); err != nil {
		return
	}

	if bo.hasTarget() {
		if err = st.New(bo.NewBlockOperationTargetKey(bo.Target), bo.Hash); err != nil {
			return
		}
		if err = st.New(bo.NewBlockOperationTargetAndTypeKey(bo.Target), bo.Hash); err != nil {
			return
		}
		if err = st.New(bo.NewBlockOperationPeersKey(bo.Target), bo.Hash); err != nil {
			return
		}
		if err = st.New(bo.NewBlockOperationPeersAndTypeKey(bo.Target), bo.Hash); err != nil {
			return
		}
	}

	if bo.targetIsLinked() {
		if err = st.New(GetBlockOperationCreateFrozenKey(bo.Target, bo.Height), bo.Hash); err != nil {
			return err
		}
		if err = st.New(bo.NewBlockOperationFrozenLinkedKey(bo.linked), bo.Hash); err != nil {
			return err
		}
	}

	bo.isSaved = true

	return nil
}

func (bo BlockOperation) Serialize() (encoded []byte, err error) {
	encoded, err = common.EncodeJSONValue(bo)
	return
}

func GetBlockOperationKey(hash string) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixHash, hash)
	return idx.String()
}

func GetBlockOperationCreateFrozenKey(hash string, height uint64) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixCreateFrozen)
	idx.WritePrefix(common.EncodeUint64ToString(height))
	idx.WritePrefix(hash)
	return idx.String()
}

func GetBlockOperationKeyPrefixFrozenLinked(hash string) string {
	idx := storage.NewIndex()
	return idx.WritePrefix(common.BlockOperationPrefixFrozenLinked, hash).String()
}

func GetBlockOperationKeyPrefixTxHash(txHash string) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixTxHash, txHash)
	return idx.String()
}

func GetBlockOperationKeyPrefixSource(source string) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixSource, source)
	return idx.String()
}

func GetBlockOperationKeyPrefixSourceAndType(source string, ty operation.OperationType) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixSource, string(ty), source)
	return idx.String()
}

func GetBlockOperationKeyPrefixBlockHeight(height uint64) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixBlockHeight)
	idx.WritePrefix(common.EncodeUint64ToString(height))
	return idx.String()
}

func GetBlockOperationKeyPrefixTarget(target string) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixTarget, target)
	return idx.String()
}

func GetBlockOperationKeyPrefixTargetAndType(target string, ty operation.OperationType) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixTypeTarget)
	idx.WritePrefix(string(ty), target)
	return idx.String()
}

func GetBlockOperationKeyPrefixPeers(addr string) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixPeers, addr)
	return idx.String()
}

func GetBlockOperationKeyPrefixPeersAndType(addr string, ty operation.OperationType) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixTypePeers)
	idx.WritePrefix(string(ty), addr)
	return idx.String()
}

func (bo BlockOperation) NewBlockOperationTxHashKey() string {
	idx := storage.NewIndex()
	idx.WritePrefix(GetBlockOperationKeyPrefixTxHash(bo.TxHash))
	bo.order.Index(idx)
	return idx.String()
	/*
		return fmt.Sprintf(
			"%s%s%s%s",
			GetBlockOperationKeyPrefixTxHash(bo.TxHash),
			common.EncodeUint64ToByteSlice(bo.Height),
			common.EncodeUint64ToByteSlice(bo.TxIndex),
			common.EncodeUint64ToByteSlice(bo.Index),
		)
	*/
}

func (bo BlockOperation) NewBlockOperationSourceKey() string {
	idx := storage.NewIndex()
	idx.WritePrefix(GetBlockOperationKeyPrefixSource(bo.Source))
	bo.order.Index(idx)
	return idx.String()
	/*
		return fmt.Sprintf(
			"%s%s%s%s",
			GetBlockOperationKeyPrefixSource(bo.Source),
			common.EncodeUint64ToByteSlice(bo.Height),
			common.EncodeUint64ToByteSlice(bo.TxIndex),
			common.EncodeUint64ToByteSlice(bo.Index),
		)
	*/
}

func (bo BlockOperation) NewBlockOperationFrozenLinkedKey(hash string) string {
	idx := storage.NewIndex()
	idx.WritePrefix(GetBlockOperationKeyPrefixFrozenLinked(hash))
	bo.order.Index(idx)
	return idx.String()

	/*
		return fmt.Sprintf(
			"%s%s",
			GetBlockOperationKeyPrefixFrozenLinked(hash),
			common.EncodeUint64ToByteSlice(bo.Height),
		)
	*/
}

func (bo BlockOperation) NewBlockOperationSourceAndTypeKey() string {
	idx := storage.NewIndex()
	idx.WritePrefix(GetBlockOperationKeyPrefixSourceAndType(bo.Source, bo.Type))
	bo.order.Index(idx)
	return idx.String()
	/*
		return fmt.Sprintf(
			"%s%s%s%s",
			GetBlockOperationKeyPrefixSourceAndType(bo.Source, bo.Type),
			common.EncodeUint64ToByteSlice(bo.Height),
			common.EncodeUint64ToByteSlice(bo.TxIndex),
			common.EncodeUint64ToByteSlice(bo.Index),
		)
	*/
}
func (bo BlockOperation) NewBlockOperationTargetKey(target string) string {
	return fmt.Sprintf(
		"%s%s%s%s",
		GetBlockOperationKeyPrefixTarget(target),
		common.EncodeUint64ToByteSlice(bo.Height),
		common.EncodeUint64ToByteSlice(bo.transaction.B.SequenceID),
		common.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationTargetAndTypeKey(target string) string {
	return fmt.Sprintf(
		"%s%s%s%s",
		GetBlockOperationKeyPrefixTargetAndType(target, bo.Type),
		common.EncodeUint64ToByteSlice(bo.Height),
		common.EncodeUint64ToByteSlice(bo.transaction.B.SequenceID),
		common.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationPeersKey(addr string) string {
	return fmt.Sprintf(
		"%s%s%s%s",
		GetBlockOperationKeyPrefixPeers(addr),
		common.EncodeUint64ToByteSlice(bo.Height),
		common.EncodeUint64ToByteSlice(bo.transaction.B.SequenceID),
		common.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationPeersAndTypeKey(addr string) string {
	return fmt.Sprintf(
		"%s%s%s%s",
		GetBlockOperationKeyPrefixPeersAndType(addr, bo.Type),
		common.EncodeUint64ToByteSlice(bo.Height),
		common.EncodeUint64ToByteSlice(bo.transaction.B.SequenceID),
		common.GetUniqueIDFromUUID(),
	)
}
func (bo BlockOperation) NewBlockOperationBlockHeightKey() string {
	idx := storage.NewIndex()
	idx.WritePrefix(GetBlockOperationKeyPrefixBlockHeight(bo.Height))
	bo.order.Index(idx)
	return idx.String()
}

func (bo BlockOperation) BlockOrder() *BlockOrder {
	return bo.order
}

func ExistsBlockOperation(st *storage.LevelDBBackend, hash string) (bool, error) {
	return st.Has(GetBlockOperationKey(hash))
}

func GetBlockOperation(st *storage.LevelDBBackend, hash string) (bo BlockOperation, err error) {
	if err = st.Get(GetBlockOperationKey(hash), &bo); err != nil {
		return
	}

	bo.isSaved = true
	bo.order = NewBlockOpOrder(bo.Height, bo.TxIndex, bo.Index)
	return
}

func LoadBlockOperationsInsideIterator(
	st *storage.LevelDBBackend,
	iterFunc func() (storage.IterItem, bool),
	closeFunc func(),
) (
	func() (BlockOperation, bool, []byte),
	func(),
) {

	return (func() (BlockOperation, bool, []byte) {
			item, hasNext := iterFunc()
			if !hasNext {
				return BlockOperation{}, false, item.Key
			}

			var hash string
			common.MustUnmarshalJSON(item.Value, &hash)

			bo, err := GetBlockOperation(st, hash)
			if err != nil {
				return BlockOperation{}, false, item.Key
			}

			return bo, hasNext, item.Key
		}), (func() {
			closeFunc()
		})
}

func GetBlockOperationsByTxHash(st *storage.LevelDBBackend, txHash string, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixTxHash(txHash), options)

	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsBySource(st *storage.LevelDBBackend, source string, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixSource(source), options)

	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

// Find all operations which created frozen account.
func GetBlockOperationsByFrozen(st *storage.LevelDBBackend, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(common.BlockOperationPrefixCreateFrozen, options)
	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

// Find all operations which created frozen account and have the link of a general account's address.
func GetBlockOperationsByLinked(st *storage.LevelDBBackend, hash string, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixFrozenLinked(hash), options)
	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsBySourceAndType(st *storage.LevelDBBackend, source string, ty operation.OperationType, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixSourceAndType(source, ty), options)
	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsByTarget(st *storage.LevelDBBackend, target string, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixTarget(target), options)

	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsByTargetAndType(st *storage.LevelDBBackend, target string, ty operation.OperationType, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixTargetAndType(target, ty), options)
	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsByPeers(st *storage.LevelDBBackend, addr string, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixPeers(addr), options)

	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsByPeersAndType(st *storage.LevelDBBackend, addr string, ty operation.OperationType, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixPeersAndType(addr, ty), options)
	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsByBlockHeight(st *storage.LevelDBBackend, height uint64, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixBlockHeight(height), options)
	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}
