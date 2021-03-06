package tools

import (
	"go.uber.org/atomic"
	"sync"
)

// Wg
type Wg struct {
	Wg    	*sync.WaitGroup 	// WaitGroup
	Count 	*atomic.Uint32      // Счётчик горутин
}

// Обертка над Wg метод Add
func (wg *Wg) Add(delta int) {
	wg.Wg.Add(1)
	wg.Count.Add(uint32(delta))
}

// Обертка над Wg метод Done
func (wg *Wg) Done() {
	wg.Wg.Done()
	wg.Count.Sub(1)
}

// Обертка над Wg метод Wait
func (wg *Wg) Wait() {
	wg.Wg.Wait()
}