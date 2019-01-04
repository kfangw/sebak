package avl

import (
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestN (t *testing.T){
	n := &N{
		Key: []byte("key"),
		Value: []byte("value"),
		Version: 1,
		Height: 2,
		Size: 3,
		Hash: []byte("hash"),
		LeftHash: []byte("leftHash"),
		RightHash: []byte("rightHash"),
	}

	b, err := rlp.EncodeToBytes(n)
	require.NoError(t, err)
	t.Log(b)

	var nn N
	rlp.DecodeBytes(b, &nn)
	t.Log(n)
	t.Log(&nn)
}
