package avl

import (
	"github.com/ethereum/go-ethereum/rlp"
	cmn "github.com/tendermint/tendermint/libs/common"
	"io"
)

type N struct {
	Key       []byte
	Value     []byte
	Version   uint64
	Height    uint8
	Size      uint64
	Hash      []byte
	LeftHash  []byte
	RightHash []byte

	leftNode  *Node `rlp:"-"`
	rightNode *Node `rlp:"-"`
	persisted bool `rlp:"-"`
}
func (node *N) isLeaf() bool {
	return node.Height == 0
}

func (node *N) EncodeRLP(w io.Writer) (err error) {
	var cause error
	cause = rlp.Encode(w, node.Height)
	if cause != nil {
		return cmn.ErrorWrap(cause, "writing height")
	}
	cause = rlp.Encode(w, node.Size)
	if cause != nil {
		return cmn.ErrorWrap(cause, "writing size")
	}
	cause = rlp.Encode(w, node.Version)
	if cause != nil {
		return cmn.ErrorWrap(cause, "writing version")
	}

	// Unlike writeHashBytes, key is written for inner nodes.
	cause = rlp.Encode(w, node.Key)
	if cause != nil {
		return cmn.ErrorWrap(cause, "writing key")
	}

	//if node.isLeaf() {
	//	cause = rlp.Encode(w, node.Value)
	//	if cause != nil {
	//		return cmn.ErrorWrap(cause, "writing value")
	//	}
	//} else {
	//	if node.LeftHash == nil {
	//		panic("node.leftHash was nil in writeBytes")
	//	}
	//	cause = rlp.Encode(w, node.LeftHash)
	//	if cause != nil {
	//		return cmn.ErrorWrap(cause, "writing left hash")
	//	}
	//
	//	if node.RightHash == nil {
	//		panic("node.rightHash was nil in writeBytes")
	//	}
	//	cause = rlp.Encode(w, node.RightHash)
	//	if cause != nil {
	//		return cmn.ErrorWrap(cause, "writing right hash")
	//	}
	//}
	return nil
}

func (node *N) DecodeRLP(s *rlp.Stream) (err error) {
	h, err := s.Uint()
	if err != nil {
		return err
	}
	node.Height = uint8(h)
	ss, err := s.Uint()
	if err != nil {
		return err
	}
	node.Size = ss
	v, err := s.Uint()
	if err != nil {
		return err
	}
	node.Version = v

	k, err := s.Bytes()
	if err != nil {
		return err
	}
	node.Key = k

	return nil
}