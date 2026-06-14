package council

import (
	"context"
	"sync"
)

// Member represents a council member capable of producing opinions.
type Member interface {
	ID() string
	Opinion(ctx context.Context, topic string) (string, error)
}

// Council manages consensus among multiple members.
type Council struct {
	mu       sync.RWMutex
	members  map[string]Member
}

// New creates an empty council.
func New() *Council {
	return &Council{members: make(map[string]Member)}
}

// Register adds a member to the council.
func (c *Council) Register(m Member) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.members[m.ID()] = m
}

// Unregister removes a member by ID.
func (c *Council) Unregister(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.members, id)
}

// Members returns a copy of the registered member IDs.
func (c *Council) Members() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ids := make([]string, 0, len(c.members))
	for id := range c.members {
		ids = append(ids, id)
	}
	return ids
}

// ConsensusResult holds each member's opinion and the final decision.
type ConsensusResult struct {
	Opinions  map[string]string `json:"opinions"`
	Decision  string            `json:"decision"`
}

// RunConsensus collects opinions from all members and selects the most common (plurality) answer.
// If no consensus, returns the first opinion.
func (c *Council) RunConsensus(ctx context.Context, topic string) (*ConsensusResult, error) {
	c.mu.RLock()
	members := make([]Member, 0, len(c.members))
	for _, m := range c.members {
		members = append(members, m)
	}
	c.mu.RUnlock()

	opinions := make(map[string]string)
	for _, m := range members {
		opinion, err := m.Opinion(ctx, topic)
		if err != nil {
			opinion = "error: " + err.Error()
		}
		opinions[m.ID()] = opinion
	}

	decision := selectDecision(opinions)
	return &ConsensusResult{Opinions: opinions, Decision: decision}, nil
}

// selectDecision picks the most frequent opinion (plurality).
// In a real system this would use more nuanced logic (e.g., weighted voting, debate rounds).
func selectDecision(opinions map[string]string) string {
	freq := make(map[string]int)
	for _, o := range opinions {
		freq[o]++
	}

	var best string
	var bestCount int
	for o, count := range freq {
		if count > bestCount {
			best = o
			bestCount = count
		}
	}

	if best == "" {
		// Fallback: first opinion
		for _, o := range opinions {
			return o
		}
	}
	return best
}
