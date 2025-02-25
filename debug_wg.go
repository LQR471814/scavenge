package scavenge

import (
	"sync"
	"sync/atomic"
)

type debugwg struct {
	wg    sync.WaitGroup
	count atomic.Int64
}

func (w *debugwg) Add(n int) {
	w.count.Add(int64(n))
	w.wg.Add(n)
}

func (w *debugwg) Done() {
	w.count.Add(-1)
	w.wg.Done()
}

func (w *debugwg) Wait() {
	w.wg.Wait()
}

func (w *debugwg) Count() int64 {
	return w.count.Load()
}
