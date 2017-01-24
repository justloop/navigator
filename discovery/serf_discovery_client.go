package discovery

import (
	"strconv"

	"github.com/hashicorp/serf/client"
	"github.com/hashicorp/serf/serf"
)

// SerfClient is the client to communicate with serf cluster
type SerfClient struct {
	c *client.RPCClient
}

// NewSerfClient will create a new client by input a cluster address
func NewSerfClient(addr string) (*SerfClient, error) {
	rpcCli, err := client.NewRPCClient(addr)
	return &SerfClient{
		c: rpcCli,
	}, err
}

// Servers will return the servers
func (s *SerfClient) Servers() ([]string, error) {
	members, err := s.c.Members()
	if err != nil {
		return []string{}, err
	}
	return getClientServers(members), nil
}

// AliveServers will return the current alive servers
func (s *SerfClient) AliveServers() ([]string, error) {
	members, err := s.c.MembersFiltered(make(map[string]string), "alive", "")
	if err != nil {
		return []string{}, err
	}
	return getClientServers(members), nil
}

// AliveServersWithTags will return the current alive servers together with its tags
func (s *SerfClient) AliveServersWithTags() (map[string]map[string]string, error) {
	result := make(map[string]map[string]string)
	allMembers, err := s.c.Members()
	if err != nil {
		return result, err
	}
	for _, member := range allMembers {
		if member.Status == serf.StatusAlive.String() {
			result[getClientServer(member)] = member.Tags
		}
	}
	return result, nil
}

// Stats will return a list of stats regarding the discovery cluster
func (s *SerfClient) Stats() (map[string]map[string]string, error) {
	return s.c.Stats()
}

// Close will close the client session
func (s *SerfClient) Close() error {
	return s.c.Close()
}

// getClientServers will return members as string slice
func getClientServers(members []client.Member) []string {
	servers := []string{}
	for _, member := range members {
		servers = append(servers, member.Addr.String()+":"+strconv.Itoa(int(member.Port)))
	}
	return servers
}

// getClientServer will convert member object to string info
func getClientServer(member client.Member) string {
	return member.Addr.String() + ":" + strconv.Itoa(int(member.Port))
}
