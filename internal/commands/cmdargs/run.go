package cmdargs

import "time"

type RunArgs struct {
	ZookeeperServers          []string
	LeaderTimeout             time.Duration
	AttempterTimeout          time.Duration
	MaxDeadLeaderTimeout      time.Duration
	FailoverQuickRetryTimeout time.Duration
	FailoverSlowRetryStep     time.Duration
	FailoverMaxStateDuration  time.Duration
	ElectionFileDir           string
	LeaderFileDir             string
	StorageCapacity           int
}
