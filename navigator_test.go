package navigator

import (
	"testing"

	"time"

	"github.com/justloop/navigator/discovery"
	"github.com/stretchr/testify/assert"
)

func TestNode(t *testing.T) {
	navigatorConfig := &Config{
		ReplicaPoints: 1,
		DConfig: &discovery.Config{
			Seed: []string{"0.0.0.0:7946"},
		},
	}
	defaultCPURefreshInterval = 100 * time.Millisecond
	node := New(navigatorConfig)
	_ = node.Start()
	defer node.Stop()
	assert.True(t, node.Ready(), "Node is not ready")

	debug := node.Debug()
	assert.Equal(t, []string{"127.0.0.1:7946"}, debug["discovery"])
	assert.Equal(t, []string{"127.0.0.1:7946"}, debug["hashring"])
	assert.Equal(t, "IndentityStrategy", debug["strategy"])
	status, _ := debug["status"].(int)
	assert.Equal(t, 0, status)

	list, err := node.GetServerList()
	assert.Nil(t, err)
	assert.Equal(t, map[string]map[string]string{"127.0.0.1:7946": {}}, list)
}
