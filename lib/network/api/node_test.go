package api

import (
	"bufio"
	"io/ioutil"
	"testing"

	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"github.com/stretchr/testify/require"
)

func TestGetNodeHandler(t *testing.T) {
	n := 3
	var ns []*network.MemoryNetwork
	var net *network.MemoryNetwork
	var nodes []*node.LocalNode
	for i := 0; i < n; i++ {
		_, s, v := network.CreateMemoryNetwork(net)
		net = s
		ns = append(ns, s)
		nodes = append(nodes, v)
	}

	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				continue
			}
			nodes[i].AddValidators(nodes[j].ConvertToValidator())
		}
	}

	ts, handler, err := prepareAPIServerWithNode(nodes[0], ns[0])
	require.Nil(t, err)
	storage := handler.storage
	defer storage.Close()
	defer ts.Close()

	handler.localNode = nodes[0]

	// Do a Request
	url := "/node"
	respBody, err := request(ts, url, false)
	require.Nil(t, err)
	defer respBody.Close()
	reader := bufio.NewReader(respBody)

	readByte, err := ioutil.ReadAll(reader)
	require.Nil(t, err)

	t.Log(string(readByte))

}
