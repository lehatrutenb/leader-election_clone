package leader_s

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/commands/cmdargs"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/ticker"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/usecases/run/states"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/usecases/run/states/failover_s"
	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/usecases/run/states/stopping_s"
	"github.com/go-zookeeper/zk"
)

func New(logger *slog.Logger, conn *zk.Conn, ticker ticker.Ticker, opts cmdargs.RunArgs) *State {
	logger = logger.With("subsystem", "LeaderState")
	return &State{
		logger:  logger,
		conn:    conn,
		ticker:  ticker,
		options: opts,
	}
}

type State struct {
	logger  *slog.Logger
	conn    *zk.Conn
	ticker  ticker.Ticker
	options cmdargs.RunArgs
}

func (s *State) String() string {
	return "LeaderState"
}

func (s *State) Int() int {
	return 2
}

func (s *State) SetZkConnection(conn *zk.Conn) {
	s.conn = conn
}

func (s *State) checkDataDir(chld []string, stat *zk.Stat) bool {
	if int(stat.NumChildren) > s.options.StorageCapacity {
		return false
	}
	for i, ch := range chld {
		if strconv.Itoa(i) != ch {
			return false
		}
	}
	return true
}

func (s *State) workWithOldData(ctx context.Context) (int, error) {
	s.logger.LogAttrs(ctx, slog.LevelDebug, "Leader working with prev leader data")
	chld, stat, err := s.conn.Children(s.options.LeaderFileDir)
	if err != nil {
		s.logger.LogAttrs(ctx, slog.LevelError, fmt.Sprint("Failed to get info about previous leader dir: ", err.Error()))
		return 0, err
	}
	if !s.checkDataDir(chld, stat) { // seems that leaders have different options or smth broken - rm old files as good tone
		s.logger.LogAttrs(ctx, slog.LevelDebug, "Leader deleting another optioned leader files")
		for _, fpth := range chld {
			if err := s.conn.Delete(s.options.LeaderFileDir+"/"+fpth, 0); err != nil {
				s.logger.LogAttrs(ctx, slog.LevelError, fmt.Sprint("Failed to delete children another version folder: ", err.Error()))
				return 0, err
			}
		}
		return 0, nil
	}
	return int(stat.NumChildren), nil
}

func (s *State) prepareLeaderFileNode(ctx context.Context) (int, error) {
	s.logger.LogAttrs(ctx, slog.LevelDebug, "Leader started prepearing its folder")
	var err error
	if _, err = s.conn.Create(s.options.LeaderFileDir, []byte{}, 0, zk.WorldACL(zk.PermAll)); err != nil && !errors.Is(err, zk.ErrNodeExists) {
		s.logger.LogAttrs(ctx, slog.LevelError, fmt.Sprint("Failed to create leader file dir: ", err.Error()))
		return 0, err
	} else if errors.Is(err, zk.ErrNodeExists) { // if already exist we should prepare it to work with
		return s.workWithOldData(ctx)
	}
	return 0, nil
}

func (s *State) Run(ctx context.Context) (states.AutomataState, error) {
	tckr, stTckr := s.ticker.GetTicker(s.options.LeaderTimeout)
	defer stTckr()

	fi, err := s.prepareLeaderFileNode(ctx)
	if err != nil {
		return failover_s.New(s.logger, s, err, s.conn, s.ticker, s.options), nil
	}

	for ; ; fi++ {
		select {
		case <-tckr:
			if fi >= s.options.StorageCapacity {
				if err := s.conn.Delete(s.options.LeaderFileDir+fmt.Sprint("/", fi%s.options.StorageCapacity), 0); err != nil {
					s.logger.LogAttrs(ctx, slog.LevelError, fmt.Sprint("Failed to delete file: ", err.Error()))
					return failover_s.New(s.logger, s, err, s.conn, s.ticker, s.options), nil
				}
			}
			if _, err := s.conn.Create(s.options.LeaderFileDir+fmt.Sprint("/", fi%s.options.StorageCapacity), []byte{}, 0, zk.WorldACL(zk.PermAll)); err != nil {
				s.logger.LogAttrs(ctx, slog.LevelError, fmt.Sprint("Failed to create file as leader: ", err.Error()))
				return failover_s.New(s.logger, s, err, s.conn, s.ticker, s.options), nil
			}
			s.logger.LogAttrs(ctx, slog.LevelDebug, "Leader created file")
		case <-ctx.Done():
			return stopping_s.New(s.logger, s.conn, ctx.Err(), s), nil
		}
	}
}
