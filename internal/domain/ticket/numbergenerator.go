package ticket

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type NumberGenerator interface {
	Generate(ctx context.Context) (string, error)
}

type DefaultNumberGenerator struct {
	mu       sync.Mutex
	counters map[string]int
}

func NewDefaultNumberGenerator() *DefaultNumberGenerator {
	return &DefaultNumberGenerator{
		counters: make(map[string]int),
	}
}

func (g *DefaultNumberGenerator) Generate(ctx context.Context) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := time.Now()
	dateKey := now.Format("20060102")

	counter, exists := g.counters[dateKey]
	if !exists {
		counter = 0
	}
	counter++
	g.counters[dateKey] = counter

	number := fmt.Sprintf("T-%s-%04d", dateKey, counter)
	return number, nil
}
