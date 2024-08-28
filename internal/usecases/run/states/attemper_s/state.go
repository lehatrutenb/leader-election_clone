package attemper_s

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/commands/cmdargs"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/ticker"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/usecases/run/states"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/usecases/run/states/failover_s"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/usecases/run/states/leader_s"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/usecases/run/states/stopping_s"
	"github.com/go-zookeeper/zk"
)

func New(logger *slog.Logger, conn *zk.Conn, ticker ticker.Ticker, opts cmdargs.RunArgs) *State {
	logger = logger.With("subsystem", "AttemperState")
	return &State{
		logger:  logger,
		conn:    conn,
		options: opts,
		ticker:  ticker,
	}
}

type State struct {
	logger  *slog.Logger
	conn    *zk.Conn
	ticker  ticker.Ticker
	options cmdargs.RunArgs
}

func (s *State) String() string {
	return "AttemperState"
}

func (s *State) Int() int {
	return 1
}

func (s *State) SetZkConnection(conn *zk.Conn) {
	s.conn = conn
}

func (s *State) Run(ctx context.Context) (states.AutomataState, error) {
	tckr, stTckr := s.ticker.GetTicker(s.options.AttempterTimeout)
	defer stTckr()
	for {
		select {
		case <-tckr:
			if _, err := s.conn.Create(s.options.ElectionFileDir, []byte{}, zk.FlagEphemeral, zk.WorldACL(zk.PermAll)); err != nil && !errors.Is(err, zk.ErrNodeExists) {
				s.logger.LogAttrs(ctx, slog.LevelError, fmt.Sprint("Got error creating znode: ", err.Error()))
				return failover_s.New(s.logger, s, err, s.conn, s.ticker, s.options), nil
			} else if errors.Is(err, zk.ErrNodeExists) {
				s.logger.LogAttrs(ctx, slog.LevelDebug, "Failed to become leader - already have another one")
				continue
			} else {
				s.logger.LogAttrs(ctx, slog.LevelInfo, "Succesfully created file as attemper")
				return leader_s.New(s.logger, s.conn, s.ticker, s.options), nil
			}
		case <-ctx.Done():
			return stopping_s.New(s.logger, s.conn, ctx.Err(), s), nil
		}
	}
}
