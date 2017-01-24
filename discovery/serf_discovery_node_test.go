package discovery

import (
	"testing"
	"time"

	"reflect"
	"sync"

	"os"

	"github.com/hashicorp/serf/cmd/serf/command/agent"
	"github.com/hashicorp/serf/serf"
	"github.com/myteksi/go/sextant/commons/utils"
	"github.com/stretchr/testify/assert"
)

var (
	port    = 9001
	rpcPort = 8001
	address = "127.0.0.1"
)

func testConfig() *Config {
	config := &Config{
		Address:    address,
		Port:       port,
		ClientAddr: address,
		ClientPort: rpcPort,
	}
	port++
	rpcPort++
	return config
}

// shouldRunMultiNodeTest will run the MultiNodeTest when required,
// it is not actually recommended to run multiple node per host,
// because it might have race condition
func shouldRunMultiNodeTest() bool {
	return os.Getenv("MULTI_NODE_TEST") == "yes"
}

var testGetAgentAndSerfConfig = func(config *Config) (*agent.Config, *serf.Config) {
	aConfig := agent.DefaultConfig()
	if len(config.ClientAddr) > 0 && config.ClientPort > 0 {
		aConfig.RPCAddr = GetClientAddrStr(config)
	}
	sConfig := serf.DefaultConfig()
	if len(config.Address) > 0 {
		sConfig.MemberlistConfig.BindAddr = config.Address
	}
	if config.Port > 0 {
		sConfig.MemberlistConfig.BindPort = config.Port
	}
	sConfig.NodeName = GetClusterAddrStr(config)
	sConfig.Init()

	// Set probe intervals that are aggressive for finding bad nodes
	sConfig.MemberlistConfig.GossipInterval = 5 * time.Millisecond
	sConfig.MemberlistConfig.ProbeInterval = 50 * time.Millisecond
	sConfig.MemberlistConfig.ProbeTimeout = 25 * time.Millisecond
	sConfig.MemberlistConfig.SuspicionMult = 1

	// Set a short reap interval so that it can run during the test
	sConfig.ReapInterval = 1 * time.Second

	// Set a short reconnect interval so that it can run a lot during tests
	sConfig.ReconnectInterval = 100 * time.Millisecond

	// Set basically zero on the reconnect/tombstone timeouts so that
	// they're removed on the first ReapInterval.
	sConfig.ReconnectTimeout = 1 * time.Microsecond
	sConfig.TombstoneTimeout = 1 * time.Microsecond

	return aConfig, sConfig
}

type testEventHandler struct {
	memberEventList []MemberEvent
	wg              *sync.WaitGroup
	handler         func(event MemberEvent) error
}

func newTestEventHandler(ignoreFail bool) *testEventHandler {
	eventHandler := &testEventHandler{
		memberEventList: []MemberEvent{},
		wg:              &sync.WaitGroup{},
	}
	eventHandler.handler = func(event MemberEvent) error {
		// we simple ignore the member failed event, because it will generate extra member event
		// moreover it's not guaranteed to be generated
		if (event.Type == EventMemberFailed || event.Type == EventMemberReap) && ignoreFail {
			return nil
		}
		eventHandler.memberEventList = append(eventHandler.memberEventList, event)
		eventHandler.wg.Done()
		return nil
	}

	return eventHandler
}

func (eventHandler *testEventHandler) string() string {
	a := ""
	for _, event := range eventHandler.memberEventList {
		a = a + "|" + utils.GetJSONStr(event)
	}
	return a
}

func (eventHandler *testEventHandler) containMemberEvent(event serf.MemberEvent) bool {
	for _, a := range eventHandler.memberEventList {
		if reflect.DeepEqual(a.Servers, GetServers(event.Members)) {
			return true
		}
	}
	return false
}

func (eventHandler *testEventHandler) memberEventCount() int {
	return len(eventHandler.memberEventList)
}

func checkShutdown(t *testing.T, node *SerfDiscoverNode) {
	err := node.Shutdown()
	if err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestJoin(t *testing.T) {
	if !shouldRunMultiNodeTest() {
		return
	}
	// mock getAgentAndSerfConfig to get faster discovery
	defer func(f func(config *Config) (*agent.Config, *serf.Config)) {
		getAgentAndSerfConfig = f
	}(getAgentAndSerfConfig)
	getAgentAndSerfConfig = testGetAgentAndSerfConfig

	listener1 := newTestEventHandler(true)
	listener1.wg.Add(4)
	s1Config := testConfig()
	s1, err := NewSerfDiscoverNodeWithHandler(s1Config, []HandlerFunc{listener1.handler})
	defer checkShutdown(t, s1)
	assert.Nil(t, err, "serf node creation failure")
	err = s1.Start()
	assert.Nil(t, err, "serf node start failure")

	listener2 := newTestEventHandler(true)
	listener2.wg.Add(4)
	s2Config := testConfig()
	s2, err := NewSerfDiscoverNodeWithHandler(s2Config, []HandlerFunc{listener2.handler})
	defer checkShutdown(t, s2)
	assert.Nil(t, err, "serf node creation failure")
	err = s2.Start()
	assert.Nil(t, err, "serf node start failure")
	_, err = s2.Join([]string{GetClusterAddrStr(s1Config)}, false)
	assert.Nil(t, err, "serf node join failure")

	listener3 := newTestEventHandler(true)
	listener3.wg.Add(4)
	s3Config := testConfig()
	s3, err := NewSerfDiscoverNodeWithHandler(s3Config, []HandlerFunc{listener3.handler})
	defer checkShutdown(t, s3)
	assert.Nil(t, err, "serf node creation failure")
	err = s3.Start()
	assert.Nil(t, err, "serf node start failure")
	_, err = s3.Join([]string{GetClusterAddrStr(s2Config)}, false)
	assert.Nil(t, err, "serf node join failure")

	listener4 := newTestEventHandler(true)
	listener4.wg.Add(4)
	s4Config := testConfig()
	s4, err := NewSerfDiscoverNodeWithHandler(s4Config, []HandlerFunc{listener4.handler})
	defer checkShutdown(t, s4)
	assert.Nil(t, err, "serf node creation failure")
	err = s4.Start()
	assert.Nil(t, err, "serf node start failure")
	_, err = s4.Join([]string{GetClusterAddrStr(s3Config)}, false)
	assert.Nil(t, err, "serf node join failure")

	listener1.wg.Wait()
	listener2.wg.Wait()
	listener3.wg.Wait()
	listener4.wg.Wait()

	// wait until shutdown event is propergated as well
	yield()
	assert.Equal(t, 4, listener1.memberEventCount(), "join event not propergated")
	assert.Equal(t, 4, listener2.memberEventCount(), "join event not propergated")
	assert.Equal(t, 4, listener3.memberEventCount(), "join event not propergated")
	assert.Equal(t, 4, listener4.memberEventCount(), "join event not propergated")

}

func TestLeave(t *testing.T) {
	if !shouldRunMultiNodeTest() {
		return
	}
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

	err = s1.Leave()
	assert.Nil(t, err, "serf node leave failure")

	listener2.wg.Wait()
	listener3.wg.Wait()

	// wait until shutdown event is propergated as well
	yield()
	assert.Equal(t, 3, listener2.memberEventCount(), "join event not propergated")
	assert.Equal(t, 3, listener3.memberEventCount(), "join event not propergated")
}

func TestSerfDiscoverNode_Tags(t *testing.T) {
	if !shouldRunMultiNodeTest() {
		return
	}
	// mock getAgentAndSerfConfig to get faster discovery
	defer func(f func(config *Config) (*agent.Config, *serf.Config)) {
		getAgentAndSerfConfig = f
	}(getAgentAndSerfConfig)
	getAgentAndSerfConfig = testGetAgentAndSerfConfig

	s1Config := testConfig()
	s1, err := NewSerfDiscoverNodeWithHandler(s1Config, []HandlerFunc{})
	assert.Nil(t, err, "serf node creation failure")
	err = s1.Start()
	assert.Nil(t, err, "serf node start failure")

	s2Config := testConfig()
	s2, err := NewSerfDiscoverNodeWithHandler(s2Config, []HandlerFunc{})
	defer checkShutdown(t, s2)
	assert.Nil(t, err, "serf node creation failure")
	err = s2.Start()
	assert.Nil(t, err, "serf node start failure")
	_, err = s2.Join([]string{GetClusterAddrStr(s1Config)}, false)
	assert.Nil(t, err, "serf node join failure")
	_ = s1.SetTags(map[string]string{})

	s3Config := testConfig()
	s3, err := NewSerfDiscoverNodeWithHandler(s3Config, []HandlerFunc{})
	defer checkShutdown(t, s3)
	assert.Nil(t, err, "serf node creation failure")
	err = s3.Start()
	assert.Nil(t, err, "serf node start failure")
	_, err = s3.Join([]string{GetClusterAddrStr(s2Config)}, false)
	assert.Nil(t, err, "serf node join failure")

	_ = s1.SetTags(map[string]string{
		"a": "b",
	})
	time.Sleep(1 * time.Second)
	assert.Equal(t, map[string]map[string]string{"127.0.0.1:9001": {"a": "b"}, "127.0.0.1:9002": {}, "127.0.0.1:9003": {}}, s1.GetServersWithTags())
	assert.Equal(t, map[string]map[string]string{"127.0.0.1:9001": {"a": "b"}, "127.0.0.1:9002": {}, "127.0.0.1:9003": {}}, s2.GetServersWithTags())
	assert.Equal(t, map[string]map[string]string{"127.0.0.1:9001": {"a": "b"}, "127.0.0.1:9002": {}, "127.0.0.1:9003": {}}, s3.GetServersWithTags())
}
