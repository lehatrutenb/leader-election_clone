package init_s

import (
	"context"
	"log/slog"

	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/commands/cmdargs"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/ticker"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/usecases/run/states"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/usecases/run/states/attemper_s"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/usecases/run/states/failover_s"
	"github.com/go-zookeeper/zk"
)

func New(logger *slog.Logger, ticker ticker.Ticker, opts cmdargs.RunArgs) *State {
	logger = logger.With("subsystem", "InitState")
	return &State{
		logger:  logger,
		options: opts,
		ticker:  ticker,
	}
}

type State struct {
	logger  *slog.Logger
	ticker  ticker.Ticker
	options cmdargs.RunArgs
}

func (s *State) String() string {
	return "InitState"
}

func (s *State) Int() int {
	return 0
}

func (s *State) SetZkConnection(_ *zk.Conn) {}

func (s *State) Run(ctx context.Context) (states.AutomataState, error) {
	conn, _, err := zk.Connect(s.options.ZookeeperServers, s.options.LeaderTimeout)
	if err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, err.Error())
		return failover_s.New(s.logger, attemper_s.New(s.logger, conn, s.ticker, s.options), err, nil, s.ticker, s.options), nil
	}
	return attemper_s.New(s.logger, conn, s.ticker, s.options), nil
}
