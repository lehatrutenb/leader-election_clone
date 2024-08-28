package stopping_s

import (
	"context"
	"log/slog"

	"github.com/central-university-dev/2024-spring-go-course-lesson8-leader-election/internal/usecases/run/states"
	"github.com/go-zookeeper/zk"
)

func New(logger *slog.Logger, conn *zk.Conn, reasonToFail error, lastState states.AutomataState) *State {
	logger = logger.With("subsystem", "StoppingState")
	return &State{
		logger:       logger,
		conn:         conn,
		reasonToFail: reasonToFail,
		lastState:    lastState,
	}
}

type State struct {
	logger       *slog.Logger
	conn         *zk.Conn
	reasonToFail error
	lastState    states.AutomataState
}

func (s *State) String() string {
	return "StoppingState"
}

func (s *State) Int() int {
	return 4
}

func (s *State) SetZkConnection(_ *zk.Conn) {}

func (s *State) Run(_ context.Context) (states.AutomataState, error) {
	if s.conn != nil {
		s.conn.Close()
	}

	return s.lastState, s.reasonToFail
}
