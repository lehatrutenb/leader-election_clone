package ticker

import (
	"time"
)

// да, можно было использовать обычную структуру time.ticker, но мне не нравится её использование только через reset

type Ticker interface {
	GetTicker(time.Duration) (ch <-chan time.Time, stop func())
	GetTimer(time.Duration) <-chan time.Time
}

type BaseTicker struct{}

func (t *BaseTicker) GetTicker(d time.Duration) (<-chan time.Time, func()) {
	tckr := time.NewTicker(d)
	return tckr.C, func() { tckr.Stop() }
}

func (t *BaseTicker) GetTimer(d time.Duration) <-chan time.Time {
	return time.NewTimer(d).C
}

func GetTicker() Ticker {
	return &BaseTicker{}
}
