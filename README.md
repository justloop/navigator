# navigator
Navigator is a distributed framework for your go service.

##
Navigator can serve as the distributed layer. It very light weighted, just contains a set of libs for distributed system

Components

	1. Discovery: Used to discover new nodes and maintains a list of alive servers follows SWIM protocol
	2. HashRing: A consistent hashring implementation
	3. Partition: A partition module using user implemented PartitionStrategy, and resolve a key to a list of servers using Hashring information
	4. Navigator: Implementation of distributed layer by combining different components.

How to use?

Configuration of navigator is easy and trivial. Configuration file is all you need to configure the navigator node:
	// Config is the configuration related to Navigator
	type Config struct {

	       // ReplicaPoints is the number of replications, optional, default 1
	       ReplicaPoints int

	       // DConfig is the configuration for discovery client, optional, default use default discovery config
	       DConfig *discovery.Config

	       // KeyHashFunc is the customized key hash function, if return 0, it will use default hash function as well
           KeyHashFunc func(key string) uint32

	       // Strategy is the partition strategy, optional, default IdentityStrategy, will resolve key to key itself and map key to a server
	       Strategy partition.Strategy

	       // SeedsService is a implementation of seedsService that used to auto get a list of seeds to join
           SeedsService seeds.Seeds

	       // OnMemberEvent is the callback handler to be invoked when repartition happens,
	       // and the node will need to reload data, optional, default, no rehashHandler
	       OnMemberEvent func(event discovery.MemberEvent) error

	       // OnShutdown is the callback when discovery is shutdown or reaped by the cluster
	       OnShutdown func()
	}

The rest all have default configuration except Strategy, application will have to extend the strategy depend on it's own scenario.

	navigatorConfig := &navigator.Config{
	       ReplicaPoints: 2,
	       Strategy:      myStrategy,
	       DConfig: &discovery.Config{
		      Address: iputil.GetHostname(),
		      Seed:    []string{}, // use seed service
	       },
	}


	n := navigator.New(navigatorConfig)
	// To start
	n.Start()


	// To stop
	n.Stop()

With navigator framework, you will be able to connect the individual nodes together to form a cluster. Then you will be able to Resolve your key to specific shard and to a list of servers holding the shard according to your partition strategy. By using the ResolveRead and ResolveWrite function

	// ResolveRead will resolve to a list of servers and partition id that this key is resolved to read from,
	// if any server read trail failed, proceed to next server in the list to read
	// return a map from shardId to servers
	ResolveRead(key string, op partition.Option) (map[string][]string, error)

	// ResolveWrite will resolve to a list of servers and partition id that this key is resolved to write to,
	// the writer will have to be responsible to write to all the servers, but ignore any fails
	// return a map from shardId to servers
	ResolveWrite(key string, op partition.Option) (map[string][]string, error)

Service Discovery

Service Discovery module is SWIM based raft consensus. Node will be able to discovery and add new nodes, and remove dead nodes by gossip protocols. More details about swim and raft: https://raft.github.io/, https://www.cs.cornell.edu/~asdas/research/dsn02-swim.pdf

Currently we use Hashicorp serf implementation of SWIM, here is the url: https://www.serf.io/

The reason of choosen that is because serf is light-weighted, and it can provide all the functions we need, just enough:

	Gossip-based membership
	Failure detection
	Custom events
	Seed service

Because SWIM based service discovery has this draw back of having to provide initial seeds for the cluster to form. We want to get rid of this for easier deployment configurations. The seed service is meant for this purpose. With seed service, we would be able to get the latest seeds, rather than provide manually.

Consistent HashRing

Consistent hashring hashes a key to a server, and moreover when server change, the number of keys affected is minimized. https://dzone.com/articles/simple-magic-consistent. The replicas of the current shard are stored in the next nodes of current node in the ring.

Partition

Partition Resolver will use the user defined PartitionStrategy to hash a key to a shardID, and find the servers that is managing the shard and return it.

Why it works?

The core idea behind the distributed framework is the HashRing replication among all the nodes. By using the service discovery module, the Rings across all cluster nodes are assumed to be the same. With the feature of consistent hashing provided by the HashRing, the shard and replicas will be consistently assigned to servers.