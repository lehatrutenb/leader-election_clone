package states

import (
	"context"

	"github.com/go-zookeeper/zk"
)

type AutomataState interface {
	Run(context.Context) (AutomataState, error)
	SetZkConnection(*zk.Conn)
	String() string
	Int() int
}
