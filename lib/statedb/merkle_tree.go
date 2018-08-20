package statedb

type MerkleTreeDB interface {
	Set(key, value []byte) error
	Get(key []byte) (value []byte, err error)
	Delete(key []byte) error

	Hash() []byte //current working root hash

	CommitDB() ([]byte, error)

	NewTreeDB(args ...interface{}) (MerkleTreeDB, error) // Return new tree which uses this tree's resources (e.g. db handle)

	Encode(interface{}) ([]byte, error)
	Decode([]byte, interface{}) error
}

//  MPTDB(eth) and IAVLDB(tender) or another merkle tree db imple.
