package db

import (
	"bytes"
	"github.com/syndtr/goleveldb/leveldb/storage"
	"path/filepath"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var _ DB = (*LevelDB)(nil)

type LevelDB struct {
	db *leveldb.DB
}

func NewLevelDB(name string, dir string) (*LevelDB, error) {
	return NewGoLevelDBWithOpts(name, dir, nil)
}

func NewMemDB() *LevelDB {
	sto := storage.NewMemStorage()
	db, err := leveldb.Open(sto, nil)
	if err != nil {
		return nil
	}
	database := &LevelDB{
		db: db,
	}
	return database
}

func NewGoLevelDBWithOpts(name string, dir string, o *opt.Options) (*LevelDB, error) {
	dbPath := filepath.Join(dir, name+".db")
	db, err := leveldb.OpenFile(dbPath, o)
	if err != nil {
		return nil, err
	}
	database := &LevelDB{
		db: db,
	}
	return database, nil
}

func (db *LevelDB) Get(key []byte) []byte {
	key = nonNilBytes(key)
	res, err := db.db.Get(key, nil)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil
		}
		panic(err)
	}
	return res
}

func (db *LevelDB) Has(key []byte) bool {
	return db.Get(key) != nil
}

func (db *LevelDB) Set(key []byte, value []byte) {
	key = nonNilBytes(key)
	value = nonNilBytes(value)
	err := db.db.Put(key, value, nil)
	if err != nil {
		panic(err)
	}
}

func (db *LevelDB) Delete(key []byte) {
	key = nonNilBytes(key)
	err := db.db.Delete(key, nil)
	if err != nil {
		panic(err)
	}
}

func (db *LevelDB) DB() *leveldb.DB {
	return db.db
}

func (db *LevelDB) Close() {
	db.db.Close()
}

func (db *LevelDB) NewBatch() Batch {
	batch := new(leveldb.Batch)
	return &levelDBBatch{db, batch}
}

func (db *LevelDB) Iterator(prefix, cursor []byte, isReverse bool) Iterator {
	var dbRange *util.Range = nil
	if len(prefix) > 0 {
		dbRange = util.BytesPrefix(prefix)
	}
	itr := db.db.NewIterator(dbRange, nil)
	return newGoLevelDBIterator(itr, cursor, isReverse)
}

//----------------------------------------
// Batch

type levelDBBatch struct {
	db    *LevelDB
	batch *leveldb.Batch
}

func (mBatch *levelDBBatch) Set(key, value []byte) {
	mBatch.batch.Put(key, value)
}

func (mBatch *levelDBBatch) Delete(key []byte) {
	mBatch.batch.Delete(key)
}

func (mBatch *levelDBBatch) Write() {
	err := mBatch.db.db.Write(mBatch.batch, &opt.WriteOptions{Sync: false})
	if err != nil {
		panic(err)
	}
}

//----------------------------------------
// Iterator

type goLevelDBIterator struct {
	source    iterator.Iterator
	isReverse bool
}

var _ Iterator = (*goLevelDBIterator)(nil)

func newGoLevelDBIterator(source iterator.Iterator, cursor []byte, isReverse bool) *goLevelDBIterator {
	if isReverse {
		if cursor == nil {
			source.Last()
		} else {
			valid := source.Seek(cursor)
			if valid {
				eoakey := source.Key() // end or after key
				if bytes.Compare(cursor, eoakey) <= 0 {
					source.Prev()
				}
			} else {
				source.Last()
			}
		}
	} else {
		if cursor == nil {
			source.First()
		} else {
			source.Seek(cursor)
		}
	}
	return &goLevelDBIterator{
		source:    source,
		isReverse: isReverse,
	}
}

// Implements Iterator.
func (itr *goLevelDBIterator) Key() []byte {
	return cp(itr.source.Key())
}

// Implements Iterator.
func (itr *goLevelDBIterator) Value() []byte {
	return cp(itr.source.Value())
}

// Implements Iterator.
func (itr *goLevelDBIterator) Next() bool {
	var exhausted bool
	if itr.isReverse {
		exhausted = itr.source.Prev()
	} else {
		exhausted = itr.source.Next()
	}
	return exhausted
}

// Implements Iterator.
func (itr *goLevelDBIterator) Close() {
	itr.source.Release()
}

func (itr *goLevelDBIterator) assertNoError() {
	if err := itr.source.Error(); err != nil {
		panic(err)
	}
}

func cp(bz []byte) (ret []byte) {
	ret = make([]byte, len(bz))
	copy(ret, bz)
	return ret
}
