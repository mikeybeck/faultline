package store

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mikey/faultline/internal/event"
)

// Result of ingesting an event.
type Result struct {
	Event   event.Event
	IsNew   bool // true when fingerprint first seen
	Updated bool
}

// Store aggregates events by fingerprint in memory.
type Store struct {
	mu     sync.RWMutex
	byHash map[string]*event.Event
	order  []string // hash order by LastSeen desc maintained on write
}

func New() *Store {
	return &Store{byHash: make(map[string]*event.Event)}
}

// Ingest merges ev into the store. Returns whether it was a new fingerprint.
func (s *Store) Ingest(ev event.Event) Result {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ev.Hash == "" {
		ev.Hash = event.Fingerprint(ev.Source, ev.Type, ev.Message, ev.File, ev.Line)
	}
	if ev.Count == 0 {
		ev.Count = 1
	}
	now := ev.Time
	if now.IsZero() {
		now = time.Now()
		ev.Time = now
	}
	if ev.FirstSeen.IsZero() {
		ev.FirstSeen = now
	}
	if ev.LastSeen.IsZero() {
		ev.LastSeen = now
	}

	existing, ok := s.byHash[ev.Hash]
	if !ok {
		clone := ev
		s.byHash[ev.Hash] = &clone
		s.order = append([]string{ev.Hash}, s.order...)
		return Result{Event: clone, IsNew: true, Updated: true}
	}

	existing.Count++
	existing.LastSeen = now
	if now.After(existing.Time) {
		existing.Time = now
	}
	// Keep richer stack/raw if newly provided.
	if ev.Stack != "" {
		existing.Stack = ev.Stack
	}
	if ev.Raw != "" {
		existing.Raw = ev.Raw
	}
	if existing.File == "" && ev.File != "" {
		existing.File = ev.File
		existing.Line = ev.Line
	}
	// Move to front of order.
	s.moveFront(ev.Hash)
	return Result{Event: *existing, IsNew: false, Updated: true}
}

func (s *Store) moveFront(hash string) {
	for i, h := range s.order {
		if h == hash {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	s.order = append([]string{hash}, s.order...)
}

// Clear removes all events.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byHash = make(map[string]*event.Event)
	s.order = nil
}

// List returns events, optionally filtered by type substring, newest last-seen first.
func (s *Store) List(filter string) []event.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filter = strings.ToLower(strings.TrimSpace(filter))
	out := make([]event.Event, 0, len(s.order))
	for _, h := range s.order {
		ev := s.byHash[h]
		if ev == nil {
			continue
		}
		if filter != "" && !matchesFilter(*ev, filter) {
			continue
		}
		out = append(out, *ev)
	}
	return out
}

// Get returns an event by hash.
func (s *Store) Get(hash string) (event.Event, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ev, ok := s.byHash[hash]
	if !ok {
		return event.Event{}, false
	}
	return *ev, true
}

// Len returns number of unique fingerprints.
func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.byHash)
}

// ByFrequency returns events sorted by count descending.
func (s *Store) ByFrequency(filter string) []event.Event {
	items := s.List(filter)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].LastSeen.After(items[j].LastSeen)
		}
		return items[i].Count > items[j].Count
	})
	return items
}

func matchesFilter(ev event.Event, filter string) bool {
	hay := strings.ToLower(strings.Join([]string{
		ev.Type, ev.Message, ev.File, ev.Source, string(ev.Severity),
	}, " "))
	return strings.Contains(hay, filter)
}
