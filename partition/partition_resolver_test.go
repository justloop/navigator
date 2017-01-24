package partition

import (
	"testing"

	"reflect"

	"github.com/stretchr/testify/assert"
)

type testRing struct{}

func newTestRing() *testRing {
	return &testRing{}
}

func (ring *testRing) Checksum() (uint32, error) {
	return 0, nil
}

func (ring *testRing) AddNode(node string) error {
	return nil
}

func (ring *testRing) RemoveNode(node string) error {
	return nil
}

func (ring *testRing) GetNodes() ([]string, error) {
	return nil, nil
}

func (ring *testRing) GetNumNodes() (int, error) {
	return 0, nil
}

func (ring *testRing) GetKeyNode(key string) (string, error) {
	return "", nil
}

// GetKeyNodes returns the all the nodes this key is hashed on according to replication level
func (ring *testRing) GetKeyNodes(key string) ([]string, error) {
	if key == "1" {
		return []string{"192.168.0.1"}, nil
	} else if key == "2" {
		return []string{"192.168.0.2"}, nil
	} else if key == "3" {
		return []string{"192.168.0.3"}, nil
	}
	return []string{"192.168.0.4"}, nil
}

func TestNewRingResolver(t *testing.T) {
	_ = NewRingResolver(newTestRing(), NewIndentityStrategy())
}

func testResolve(t *testing.T, isRead bool) {
	resolver := NewRingResolver(newTestRing(), NewIndentityStrategy())
	var (
		partitions map[string][]string
		err        error
	)
	if isRead {
		partitions, err = resolver.ResolveRead("1", nil)
		assert.Nil(t, err, "there is error in ResolveRead")
	} else {
		partitions, err = resolver.ResolveWrite("1", nil)
		assert.Nil(t, err, "there is error in ResolveWrite")
	}

	expected := make(map[string][]string)
	expected["192.168.0.1"] = []string{"1"}
	if !reflect.DeepEqual(partitions, expected) {
		t.Fatalf("expected partitions: %v. Got: %v", expected, partitions)
	}

	if isRead {
		partitions, err = resolver.ResolveRead("2", nil)
		assert.Nil(t, err, "there is error in ResolveRead")
	} else {
		partitions, err = resolver.ResolveWrite("2", nil)
		assert.Nil(t, err, "there is error in ResolveWrite")
	}
	delete(expected, "192.168.0.1")
	expected["192.168.0.2"] = []string{"2"}
	if !reflect.DeepEqual(partitions, expected) {
		t.Fatalf("expected partitions: %v. Got: %v", expected, partitions)
	}
}

func TestRingResolver_ResolveRead(t *testing.T) {
	testResolve(t, true)
}

func TestRingResolver_ResolveWrite(t *testing.T) {
	testResolve(t, false)
}
