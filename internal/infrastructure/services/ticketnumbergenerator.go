package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

type TicketNumberGenerator struct {
	db    *gorm.DB
	mu    sync.Mutex
	cache map[string]int
}

func NewTicketNumberGenerator(db *gorm.DB) *TicketNumberGenerator {
	return &TicketNumberGenerator{
		db:    db,
		cache: make(map[string]int),
	}
}

func (g *TicketNumberGenerator) Generate(ctx context.Context) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	dateStr := time.Now().Format("20060102")
	prefix := fmt.Sprintf("T-%s-", dateStr)

	seq, err := g.getNextSequence(ctx, dateStr)
	if err != nil {
		return "", err
	}

	number := fmt.Sprintf("%s%04d", prefix, seq)
	return number, nil
}

func (g *TicketNumberGenerator) getNextSequence(ctx context.Context, dateStr string) (int, error) {
	if seq, ok := g.cache[dateStr]; ok {
		g.cache[dateStr] = seq + 1
		return seq + 1, nil
	}

	var maxNumber string
	prefix := fmt.Sprintf("T-%s-%%", dateStr)

	err := g.db.WithContext(ctx).
		Table("tickets").
		Select("MAX(number)").
		Where("number LIKE ?", prefix).
		Scan(&maxNumber).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return 0, fmt.Errorf("failed to get max ticket number: %w", err)
	}

	seq := 1
	if maxNumber != "" {
		fmt.Sscanf(maxNumber, prefix[:len(prefix)-1]+"%d", &seq)
		seq++
	}

	g.cache[dateStr] = seq
	return seq, nil
}
