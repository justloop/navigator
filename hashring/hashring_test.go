package hashring

import (
	"testing"

	"math/rand"
	"strconv"

	"reflect"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	_ = NewMapHashRing(5)
}

func TestAddNode(t *testing.T) {
	mapRing := NewMapHashRing(5)
	err := mapRing.AddNode("192.168.0.1")
	assert.Nil(t, err, "add MapRing return error")

	num, err := mapRing.GetNumNodes()
	assert.Nil(t, err, "getNumNodes error")
	assert.Equal(t, 1, num, "mapRing remove server failed")
}

func TestRemoveNode(t *testing.T) {
	mapRing := NewMapHashRing(5)
	err := mapRing.AddNode("192.168.0.1")
	assert.Nil(t, err, "create MapRing return error")

	err = mapRing.RemoveNode("192.168.0.1")
	assert.Nil(t, err, "remove MapRing return error")
	num, err := mapRing.GetNumNodes()
	assert.Nil(t, err, "getNumNodes error")
	assert.Equal(t, 0, num, "mapRing remove server failed")
}

func TestLookup(t *testing.T) {
	mapRing := NewMapHashRing(2)
	err := mapRing.AddNode("192.168.0.1")
	assert.Nil(t, err, "add MapRing return error")
	err = mapRing.AddNode("192.168.0.2")
	assert.Nil(t, err, "add MapRing return error")
	err = mapRing.AddNode("192.168.0.3")
	assert.Nil(t, err, "add MapRing return error")

	nodes, err := mapRing.GetKeyNodes("test")
	assert.Nil(t, err, "GetKeyNodes return error")
	assert.Equal(t, 2, len(nodes), "GetKeyNodes failed")

	node, err := mapRing.GetKeyNode("test")
	assert.Nil(t, err, "GetKeyNodes return error")
	assert.NotEqual(t, "", node, "GetKeyNodes failed")
}

func shuffle(a []string) {
	for i := range a {
		j := rand.Intn(i + 1)
		a[i], a[j] = a[j], a[i]
	}
}

func TestConsistent(t *testing.T) {
	nodeCounts := []int{3, 5, 10, 20}
	for _, nodeCount := range nodeCounts {
		ring1 := NewMapHashRing(2)
		ring2 := NewMapHashRing(2)
		nodes := []string{}
		for n := 0; n < nodeCount; n++ {
			nodeAddr := "10.10.3." + strconv.Itoa(n) + ":7496"
			nodes = append(nodes, nodeAddr)
		}
		shuffle(nodes)
		for _, node := range nodes {
			_ = ring1.AddNode(node)
		}
		shuffle(nodes)
		for _, node := range nodes {
			_ = ring2.AddNode(node)
		}

		ring1Nodes, _ := ring1.GetNodes()
		ring2Nodes, _ := ring2.GetNodes()
		if reflect.DeepEqual(ring1Nodes, ring2Nodes) {
			_ = ring1.RemoveNode("10.10.3.0:7496")
			_ = ring1.AddNode("10.10.3.0:7496")
		}

		for i := 0; i < 300; i++ {
			for j := 0; j < 300; j++ {
				server1, _ := ring1.GetKeyNode(strconv.Itoa(i) + "," + strconv.Itoa(j))
				server2, _ := ring2.GetKeyNode(strconv.Itoa(i) + "," + strconv.Itoa(j))
				assert.Equal(t, server1, server2)
			}
		}
	}
}

func BenchmarkMapHashRing_GetKeyNodes(b *testing.B) {
	b.StopTimer()
	ring := NewMapHashRing(3)
	for n := 0; n < 100; n++ {
		nodeAddr := "10.10.3." + strconv.Itoa(n) + ":7496"
		_ = ring.AddNode(nodeAddr)
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		key := "test" + strconv.Itoa(i)
		_, _ = ring.GetKeyNodes(key)
	}
}
