package failover_s

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/commands/cmdargs"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/ticker"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/usecases/run/states"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/usecases/run/states/stopping_s"
	"github.com/go-zookeeper/zk"
)

func New(logger *slog.Logger, lastState states.AutomataState, reasonToFail error, conn *zk.Conn, ticker ticker.Ticker, opts cmdargs.RunArgs) *State {
	logger = logger.With("subsystem", "FailoverState")
	return &State{
		logger:       logger,
		lastState:    lastState,
		reasonToFail: reasonToFail,
		conn:         conn,
		ticker:       ticker,
		options:      opts,
	}
}

type State struct {
	logger       *slog.Logger
	lastState    states.AutomataState
	reasonToFail error
	conn         *zk.Conn
	ticker       ticker.Ticker
	options      cmdargs.RunArgs
}

func (s *State) String() string {
	return "FailoverState"
}

func (s *State) Int() int {
	return 3
}

func (s *State) SetZkConnection(_ *zk.Conn) {}

func (s *State) tryConnect(ctx context.Context) states.AutomataState {
	conn, _, err := zk.Connect(s.options.ZookeeperServers, s.options.LeaderTimeout)
	if err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, fmt.Sprint("Tried to reconnect failed", err))
		return nil
	}
	s.lastState.SetZkConnection(conn)
	return s.lastState
}

func (s *State) Run(ctx context.Context) (states.AutomataState, error) {
	if !errors.Is(s.reasonToFail, zk.ErrConnectionClosed) && !errors.Is(s.reasonToFail, zk.ErrSessionExpired) && !errors.Is(s.reasonToFail, zk.ErrNoServer) {
		return stopping_s.New(s.logger, s.conn, s.reasonToFail, s.lastState), nil
	}
	if s.conn != nil {
		s.conn.Close()
	}

	tckr, stTckr := s.ticker.GetTicker(s.options.FailoverQuickRetryTimeout)
	defer stTckr()
	endQTckr, stTckr := s.ticker.GetTicker(s.options.MaxDeadLeaderTimeout - s.options.FailoverQuickRetryTimeout)
	defer stTckr()
	endStateTckr, stTckr := s.ticker.GetTicker(s.options.FailoverMaxStateDuration)
	defer stTckr()
	tckrDur := -1
	for {
		select {
		case <-tckr:
			if nSt := s.tryConnect(ctx); nSt != nil {
				return nSt, nil
			}
			if tckrDur != -1 {
				tckrDur += int(s.options.FailoverSlowRetryStep)
				tckr = s.ticker.GetTimer(time.Duration(tckrDur))
			}
		case <-endQTckr:
			tckrDur = 0
			s.logger.LogAttrs(ctx, slog.LevelDebug, "Failover end quick attempts")
		case <-endStateTckr:
			s.logger.LogAttrs(ctx, slog.LevelDebug, "Failover end slow and quick attempts")
			return stopping_s.New(s.logger, nil, s.reasonToFail, s.lastState), nil
		case <-ctx.Done():
			return stopping_s.New(s.logger, nil, ctx.Err(), s.lastState), nil
		}
	}
}
