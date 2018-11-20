package block

import (
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
)

func TestNewBlockTransaction(t *testing.T) {
	conf := common.NewTestConfig()
	_, tx := transaction.TestMakeTransaction(conf.NetworkID, 1)
	block := TestMakeNewBlock([]string{tx.GetHash()})
	bt := NewBlockTransactionFromTransaction(block.Hash, block.Height, block.ProposedTime, tx, 1)

	require.Equal(t, bt.Hash, tx.H.Hash)
	require.Equal(t, bt.SequenceID, tx.B.SequenceID)
	require.Equal(t, bt.Signature, tx.H.Signature)
	require.Equal(t, bt.Source, tx.B.Source)
	require.Equal(t, bt.Fee, tx.B.Fee)
	require.Equal(t, bt.Created, tx.H.Created)

	var opHashes []string
	for i, op := range tx.B.Operations {
		opHashes = append(opHashes, NewBlockOperationKey(op.MakeHashString(), tx.GetHash(), uint64(i)))
	}
	for i, opHash := range bt.Operations {
		require.Equal(t, opHash, opHashes[i])
	}
	require.Equal(t, bt.Amount, tx.TotalAmount(true))

	for i, txHash := range block.Transactions {
		require.Equal(t, txHash, bt.Hash)
		require.Equal(t, uint64(i+1), bt.Index)
	}
}

func TestBlockTransactionSaveAndGet(t *testing.T) {
	conf := common.NewTestConfig()
	st := storage.NewTestStorage()

	bt := makeNewBlockTransaction(conf.NetworkID, 1, 0)
	err := bt.Save(st)
	require.NoError(t, err)

	fetched, err := GetBlockTransaction(st, bt.Hash)
	require.NoError(t, err)

	require.Equal(t, bt.Hash, fetched.Hash)
	require.Equal(t, bt.SequenceID, fetched.SequenceID)
	require.Equal(t, bt.Signature, fetched.Signature)
	require.Equal(t, bt.Source, fetched.Source)
	require.Equal(t, bt.Fee, fetched.Fee)
	require.Equal(t, bt.Created, fetched.Created)
	require.Equal(t, bt.Operations, fetched.Operations)
	require.Equal(t, len(fetched.Confirmed) > 0, true)
}

func TestBlockTransactionSaveExisting(t *testing.T) {
	conf := common.NewTestConfig()
	st := storage.NewTestStorage()

	bt := makeNewBlockTransaction(conf.NetworkID, 1, 0)
	err := bt.Save(st)
	require.NoError(t, err)

	exists, err := ExistsBlockTransaction(st, bt.Hash)
	require.NoError(t, err)
	require.Equal(t, exists, true)

	err = bt.Save(st)
	require.Error(t, err)
	require.Equal(t, err, errors.AlreadySaved)
}

func TestMultipleBlockTransactionSource(t *testing.T) {
	conf := common.NewTestConfig()
	kp := keypair.Random()
	kpAnother := keypair.Random()
	st := storage.NewTestStorage()

	numTxs := 10

	var txs []transaction.Transaction
	var txHashes []string
	var createdOrder []string
	for i := 0; i < numTxs; i++ {
		tx := transaction.TestMakeTransactionWithKeypair(conf.NetworkID, 1, kp)
		txs = append(txs, tx)
		createdOrder = append(createdOrder, tx.GetHash())
		txHashes = append(txHashes, tx.GetHash())

	}

	block := TestMakeNewBlock(txHashes)
	for i, tx := range txs {
		bt := NewBlockTransactionFromTransaction(block.Hash, block.Height, block.ProposedTime, tx, uint64(i))
		err := bt.Save(st)
		require.NoError(t, err)
	}

	// create txs from another keypair
	txs = []transaction.Transaction{}
	txHashes = []string{}
	for i := 0; i < numTxs; i++ {
		tx := transaction.TestMakeTransactionWithKeypair(conf.NetworkID, 1, kpAnother)
		txs = append(txs, tx)
		txHashes = append(txHashes, tx.GetHash())
	}

	block = TestMakeNewBlockWithPrevBlock(block, txHashes)
	for i, tx := range txs {
		bt := NewBlockTransactionFromTransaction(block.Hash, block.Height, block.ProposedTime, tx, uint64(i))
		err := bt.Save(st)
		require.NoError(t, err)
	}

	{
		var saved []BlockTransaction
		iterFunc, closeFunc := GetBlockTransactionsBySource(st, kp.Address(), nil)
		for {
			bo, hasNext, _ := iterFunc()
			if !hasNext {
				break
			}

			saved = append(saved, bo)
		}
		closeFunc()

		require.Equal(t, len(saved), len(createdOrder))
		for i, bt := range saved {
			require.Equal(t, createdOrder[i], bt.Hash)
		}
	}

	{
		// reverse order
		var saved []BlockTransaction
		iterFunc, closeFunc := GetBlockTransactionsBySource(st, kp.Address(), storage.NewDefaultListOptions(true, nil, uint64(numTxs)))
		for {
			bo, hasNext, _ := iterFunc()
			if !hasNext {
				break
			}

			saved = append(saved, bo)
		}
		closeFunc()

		reverseCreatedOrder := common.ReverseStringSlice(createdOrder)

		require.Equal(t, len(saved), len(createdOrder))
		for i, bt := range saved {
			require.Equal(t, reverseCreatedOrder[i], bt.Hash)
		}
	}
}

func TestMultipleBlockTransactionConfirmed(t *testing.T) {
	conf := common.NewTestConfig()
	kp := keypair.Random()
	st := storage.NewTestStorage()

	numTxs := 10

	var txs []transaction.Transaction
	var txHashes []string
	var createdOrder []string
	for i := 0; i < numTxs; i++ {
		tx := transaction.TestMakeTransactionWithKeypair(conf.NetworkID, 1, kp)
		createdOrder = append(createdOrder, tx.GetHash())
		txs = append(txs, tx)
		txHashes = append(txHashes, tx.GetHash())
	}

	block := TestMakeNewBlock(txHashes)
	for i, tx := range txs {
		bt := NewBlockTransactionFromTransaction(block.Hash, block.Height, block.ProposedTime, tx, uint64(i))
		err := bt.Save(st)
		require.NoError(t, err)
	}

	var saved []BlockTransaction
	iterFunc, closeFunc := GetBlockTransactionsByConfirmed(st, nil)
	for {
		bo, hasNext, _ := iterFunc()
		if !hasNext {
			break
		}

		saved = append(saved, bo)
	}
	closeFunc()

	require.Equal(t, len(saved), len(createdOrder))
	for i, bt := range saved {
		require.Equal(t, createdOrder[i], bt.Hash)
	}

	{
		// reverse order
		var saved []BlockTransaction
		iterFunc, closeFunc := GetBlockTransactionsByConfirmed(st, storage.NewDefaultListOptions(true, nil, uint64(numTxs)))
		for {
			bo, hasNext, _ := iterFunc()
			if !hasNext {
				break
			}

			saved = append(saved, bo)
		}
		closeFunc()

		reverseCreatedOrder := common.ReverseStringSlice(createdOrder)

		require.Equal(t, len(saved), len(createdOrder))
		for i, bt := range saved {
			require.Equal(t, reverseCreatedOrder[i], bt.Hash)
		}
	}
}

func TestBlockTransactionMultipleSave(t *testing.T) {
	conf := common.NewTestConfig()
	st := storage.NewTestStorage()

	bt := makeNewBlockTransaction(conf.NetworkID, 1, 0)
	err := bt.Save(st)
	require.NoError(t, err)

	if err = bt.Save(st); err != nil {
		if err != errors.AlreadySaved {
			t.Errorf("mutiple saving will occur error, 'ErrorAlreadySaved': %v", err)
			return
		}
	}
}

func TestMultipleBlockTransactionGetByAccount(t *testing.T) {
	conf := common.NewTestConfig()
	kp := keypair.Random()
	kpAnother := keypair.Random()
	st := storage.NewTestStorage()

	numTxs := 5

	var txs []transaction.Transaction
	var txHashes []string
	var createdOrder []string
	var blk Block
	{
		for i := 0; i < numTxs; i++ {
			tx := transaction.TestMakeTransactionWithKeypair(conf.NetworkID, 1, kp)
			txs = append(txs, tx)
			createdOrder = append(createdOrder, tx.GetHash())
			txHashes = append(txHashes, tx.GetHash())
		}

		blk = TestMakeNewBlock(txHashes)
		for i, tx := range txs {
			bt := NewBlockTransactionFromTransaction(blk.Hash, blk.Height, blk.ProposedTime, tx, uint64(i))
			bt.MustSave(st)
			err := bt.SaveBlockOperations(st)
			require.NoError(t, err)
		}
	}

	{
		// create txs from another keypair source but target is this keypair
		txs = []transaction.Transaction{}
		txHashes = []string{}
		for i := 0; i < numTxs; i++ {
			tx := transaction.TestMakeTransactionWithKeypair(conf.NetworkID, 1, kpAnother, kp)
			txs = append(txs, tx)
			createdOrder = append(createdOrder, tx.GetHash())
			txHashes = append(txHashes, tx.GetHash())
		}

		blk = TestMakeNewBlockWithPrevBlock(blk, txHashes)
		for i, tx := range txs {
			bt := NewBlockTransactionFromTransaction(blk.Hash, blk.Height, blk.ProposedTime, tx, uint64(i))
			bt.MustSave(st)
			err := bt.SaveBlockOperations(st)
			require.NoError(t, err)
		}
	}

	{
		// create txs from another keypair
		txs = []transaction.Transaction{}
		txHashes = []string{}
		for i := 0; i < numTxs; i++ {
			tx := transaction.TestMakeTransactionWithKeypair(conf.NetworkID, 1, kpAnother)
			txs = append(txs, tx)
			txHashes = append(txHashes, tx.GetHash())
		}

		blk = TestMakeNewBlockWithPrevBlock(blk, txHashes)
		for i, tx := range txs {
			bt := NewBlockTransactionFromTransaction(blk.Hash, blk.Height, blk.ProposedTime, tx, uint64(i))
			bt.MustSave(st)
			err := bt.SaveBlockOperations(st)
			require.NoError(t, err)
		}
	}

	{
		var saved []BlockTransaction
		iterFunc, closeFunc := GetBlockTransactionsByAccount(st, kp.Address(), nil)
		for {
			bo, hasNext, _ := iterFunc()
			if !hasNext {
				break
			}

			saved = append(saved, bo)
		}
		closeFunc()

		require.Equal(t, len(saved), len(createdOrder))
		for i, bt := range saved {
			require.Equal(t, bt.Hash, createdOrder[i])
		}
	}
}

func TestMultipleBlockTransactionGetByBlock(t *testing.T) {
	conf := common.NewTestConfig()
	kp := keypair.Random()
	st := storage.NewTestStorage()

	numTxs := 5

	var txs0 []transaction.Transaction
	var txHashes0 []string
	var createdOrder0 []string
	for i := 0; i < numTxs; i++ {
		tx := transaction.TestMakeTransactionWithKeypair(conf.NetworkID, 1, kp)
		txs0 = append(txs0, tx)
		createdOrder0 = append(createdOrder0, tx.GetHash())
		txHashes0 = append(txHashes0, tx.GetHash())
	}

	block0 := TestMakeNewBlock(txHashes0)
	for i, tx := range txs0 {
		bt := NewBlockTransactionFromTransaction(block0.Hash, block0.Height, block0.ProposedTime, tx, uint64(i))
		bt.MustSave(st)
	}

	var txs1 []transaction.Transaction
	var txHashes1 []string
	var createdOrder1 []string
	for i := 0; i < numTxs; i++ {
		tx := transaction.TestMakeTransactionWithKeypair(conf.NetworkID, 1, kp)
		txs1 = append(txs1, tx)
		createdOrder1 = append(createdOrder1, tx.GetHash())
		txHashes1 = append(txHashes1, tx.GetHash())
	}

	block1 := TestMakeNewBlockWithPrevBlock(block0, txHashes1)
	for i, tx := range txs1 {
		bt := NewBlockTransactionFromTransaction(block1.Hash, block1.Height, block1.ProposedTime, tx, uint64(i))
		bt.MustSave(st)
	}

	{
		var saved []BlockTransaction
		iterFunc, closeFunc := GetBlockTransactionsByBlock(st, block0.Hash, nil)
		for {
			bo, hasNext, _ := iterFunc()
			if !hasNext {
				break
			}

			saved = append(saved, bo)
		}
		closeFunc()

		require.Equal(t, len(saved), len(createdOrder0), "fetched records insufficient")
		for i, bt := range saved {
			require.Equal(t, bt.Hash, createdOrder0[i], "order mismatch")
			require.Equal(t, bt.Index, uint64(i))
		}
	}

	{
		var saved []BlockTransaction
		iterFunc, closeFunc := GetBlockTransactionsByBlock(st, block1.Hash, nil)
		for {
			bo, hasNext, _ := iterFunc()
			if !hasNext {
				break
			}

			saved = append(saved, bo)
		}
		closeFunc()

		require.Equal(t, len(saved), len(createdOrder1), "fetched records insufficient")
		for i, bt := range saved {
			require.Equal(t, bt.Hash, createdOrder1[i], "order mismatch")
		}
	}
}

func TestMultipleBlockTransactionsOrderByBlockHeightAndCursor(t *testing.T) {
	conf := common.NewTestConfig()
	kp := keypair.Random()
	st := storage.NewTestStorage()

	numTxs := 5

	// To check iteration order by height
	var transactionOrder []string

	// Make transactions with height 2 first
	{
		var createdOrder []string
		txs := []transaction.Transaction{}
		txHashes := []string{}
		for i := 0; i < numTxs; i++ {
			tx := transaction.TestMakeTransactionWithKeypair(conf.NetworkID, 1, kp)
			txs = append(txs, tx)
			createdOrder = append(createdOrder, tx.GetHash())
			txHashes = append(txHashes, tx.GetHash())
		}

		block := TestMakeNewBlock(txHashes)
		block.Height++
		for i, tx := range txs {
			bt := NewBlockTransactionFromTransaction(block.Hash, block.Height, block.ProposedTime, tx, uint64(i))
			bt.MustSave(st)
		}
		transactionOrder = append(transactionOrder, createdOrder...)
		block.MustSave(st)
	}

	// Make transactions with height 1
	{
		var createdOrder []string
		txs := []transaction.Transaction{}
		txHashes := []string{}
		for i := 0; i < numTxs; i++ {
			tx := transaction.TestMakeTransactionWithKeypair(conf.NetworkID, 1, kp)
			txs = append(txs, tx)
			createdOrder = append(createdOrder, tx.GetHash())
			txHashes = append(txHashes, tx.GetHash())
		}

		block := TestMakeNewBlock(txHashes)
		for i, tx := range txs {
			bt := NewBlockTransactionFromTransaction(block.Hash, block.Height, block.ProposedTime, tx, uint64(i))
			bt.MustSave(st)
		}

		transactionOrder = append(createdOrder, transactionOrder...)
		block.MustSave(st)
	}

	var halfSaved []BlockTransaction
	var theCursor []byte
	// Check transaction order by block height
	{
		var saved []BlockTransaction
		var cursors [][]byte
		iterFunc, closeFunc := GetBlockTransactionsByAccount(st, kp.Address(), nil)
		for {
			bo, hasNext, cursor := iterFunc()
			if !hasNext {
				break
			}
			cc := make([]byte, len(cursor))
			copy(cc, cursor)
			cursors = append(cursors, cc)
			saved = append(saved, bo)
		}
		closeFunc()

		require.Equal(t, len(saved), len(transactionOrder))
		for i, bt := range saved {
			require.Equal(t, bt.Hash, transactionOrder[i])
		}

		halfSaved = saved[len(saved)/2:]
		theCursor = cursors[len(saved)/2-1]
	}

	// Check transactions filtered by cursor
	{
		var saved []BlockTransaction
		iterFunc, closeFunc := GetBlockTransactionsByAccount(st, kp.Address(), storage.NewDefaultListOptions(false, theCursor, uint64(numTxs)))
		for {
			bo, hasNext, _ := iterFunc()
			if !hasNext {
				break
			}

			saved = append(saved, bo)
		}
		closeFunc()

		require.Equal(t, len(halfSaved), len(saved))
		for i, bt := range saved {
			require.Equal(t, bt.Hash, halfSaved[i].Hash)
		}
	}
}

func makeNewBlockTransaction(networkID []byte, n int, index uint64) BlockTransaction {
	_, tx := transaction.TestMakeTransaction(networkID, n)

	block := TestMakeNewBlock([]string{tx.GetHash()})
	return NewBlockTransactionFromTransaction(block.Hash, block.Height, block.ProposedTime, tx, index)
}
