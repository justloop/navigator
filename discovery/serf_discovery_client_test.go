package discovery

import (
	"testing"

	"reflect"

	"sort"
	"time"

	"github.com/hashicorp/serf/cmd/serf/command/agent"
	"github.com/hashicorp/serf/serf"
	"github.com/stretchr/testify/assert"
)

func yield() {
	time.Sleep(1 * time.Second)
}

// Make sure the underline client usage is correct
func testClientOutput(t *testing.T, client Client, expectedServers []string, expectedAlive []string) {
	sort.Strings(expectedServers)
	sort.Strings(expectedAlive)
	actual, err := client.AliveServers()
	sort.Strings(actual)
	assert.Nil(t, err, "Error getting alive servers")
	if !reflect.DeepEqual(expectedAlive, actual) {
		t.Fatalf("expected alive servers: %v. Got: %v", expectedAlive, actual)
	}
	actual, err = client.Servers()
	sort.Strings(actual)
	assert.Nil(t, err, "Error getting servers")
	if !reflect.DeepEqual(expectedServers, actual) {
		t.Fatalf("expected servers: %v. Got: %v", expectedServers, actual)
	}
}

func checkAndClose(t *testing.T, c Client) {
	err := c.Close()
	assert.Nil(t, err, "Close client has error")
}

// TestClient tests the client
func TestClient(t *testing.T) {
	// mock getAgentAndSerfConfig to get faster discovery
	defer func(f func(config *Config) (*agent.Config, *serf.Config)) {
		getAgentAndSerfConfig = f
	}(getAgentAndSerfConfig)
	getAgentAndSerfConfig = testGetAgentAndSerfConfig

	listener1 := newTestEventHandler(true)
	listener1.wg.Add(3)
	s1Config := testConfig()
	s1, err := NewSerfDiscoverNodeWithHandler(s1Config, []HandlerFunc{listener1.handler})
	assert.Nil(t, err, "serf node creation failure")
	err = s1.Start()
	assert.Nil(t, err, "serf node start failure")

	client, err := NewSerfClient(GetClientAddrStr(s1Config))
	defer checkAndClose(t, client)
	assert.Nil(t, err, "serf client start failure")
	expected := []string{GetClusterAddrStr(s1Config)}
	testClientOutput(t, client, expected, expected)

	listener2 := newTestEventHandler(true)
	listener2.wg.Add(3)
	s2Config := testConfig()
	s2, err := NewSerfDiscoverNodeWithHandler(s2Config, []HandlerFunc{listener2.handler})
	defer checkShutdown(t, s2)
	assert.Nil(t, err, "serf node creation failure")
	err = s2.Start()
	assert.Nil(t, err, "serf node start failure")
	_, err = s2.Join([]string{GetClusterAddrStr(s1Config)}, false)
	assert.Nil(t, err, "serf node join failure")

	client2, err := NewSerfClient(GetClientAddrStr(s1Config))
	defer checkAndClose(t, client2)
	assert.Nil(t, err, "serf client start failure")
	expected2 := []string{GetClusterAddrStr(s1Config), GetClusterAddrStr(s2Config)}
	yield()
	testClientOutput(t, client2, expected2, expected2)

	listener3 := newTestEventHandler(true)
	listener3.wg.Add(3)
	s3Config := testConfig()
	s3, err := NewSerfDiscoverNodeWithHandler(s3Config, []HandlerFunc{listener3.handler})
	defer checkShutdown(t, s3)
	assert.Nil(t, err, "serf node creation failure")
	err = s3.Start()
	assert.Nil(t, err, "serf node start failure")
	_, err = s3.Join([]string{GetClusterAddrStr(s2Config)}, false)
	assert.Nil(t, err, "serf node join failure")
	expected3 := []string{GetClusterAddrStr(s1Config), GetClusterAddrStr(s2Config), GetClusterAddrStr(s3Config)}
	yield()
	testClientOutput(t, client2, expected3, expected3)

	err = s1.Leave()
	assert.Nil(t, err, "serf node leave failure")

	expected4 := []string{GetClusterAddrStr(s2Config), GetClusterAddrStr(s3Config)}
	yield()
	testClientOutput(t, client2, expected4, expected4)

	listener2.wg.Wait()
	listener3.wg.Wait()
	yield()
}
