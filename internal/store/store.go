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
	ev.Samples = event.PushSample(ev.Samples, ev.Message)
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
	if ev.Snippet != "" {
		existing.Snippet = ev.Snippet
	}
	if len(ev.Context) > 0 {
		existing.Context = append([]string{}, ev.Context...)
	}
	existing.Message = ev.Message
	existing.Samples = event.PushSample(existing.Samples, ev.Message)
	if existing.File == "" && ev.File != "" {
		existing.File = ev.File
		existing.Line = ev.Line
	}
	return Result{Event: *existing, IsNew: false, Updated: true}
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
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].LastSeen.After(out[j].LastSeen)
	})
	return out
}

// Summaries is List without Stack/Raw, for cheap inbox snapshots.
func (s *Store) Summaries(filter string) []event.Event {
	items := s.List(filter)
	for i := range items {
		items[i].Stack = ""
		items[i].Raw = ""
		items[i].Snippet = ""
		items[i].Context = nil
		items[i].Samples = nil
	}
	return items
}

// CountsBySource returns the number of fingerprints per source.
func (s *Store) CountsBySource() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]int, 4)
	for _, ev := range s.byHash {
		if ev == nil {
			continue
		}
		out[ev.Source]++
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

// Put replaces the event for ev.Hash without incrementing count.
func (s *Store) Put(ev event.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ev.Hash == "" {
		ev.Hash = event.Fingerprint(ev.Source, ev.Type, ev.Message, ev.File, ev.Line)
	}
	clone := ev
	if _, ok := s.byHash[ev.Hash]; !ok {
		s.order = append([]string{ev.Hash}, s.order...)
	}
	s.byHash[ev.Hash] = &clone
}

// Load replaces the store with events (used when hydrating a project).
func (s *Store) Load(events []event.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byHash = make(map[string]*event.Event, len(events))
	s.order = make([]string, 0, len(events))
	for _, ev := range events {
		if ev.Hash == "" {
			ev.Hash = event.Fingerprint(ev.Source, ev.Type, ev.Message, ev.File, ev.Line)
		}
		clone := ev
		s.byHash[ev.Hash] = &clone
		s.order = append(s.order, ev.Hash)
	}
}

// Remove deletes fingerprints from the inbox.
func (s *Store) Remove(hashes ...string) {
	if len(hashes) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	drop := make(map[string]struct{}, len(hashes))
	for _, h := range hashes {
		if h == "" {
			continue
		}
		drop[h] = struct{}{}
		delete(s.byHash, h)
	}
	if len(drop) == 0 {
		return
	}
	kept := s.order[:0]
	for _, h := range s.order {
		if _, ok := drop[h]; !ok {
			kept = append(kept, h)
		}
	}
	s.order = kept
}

// RemoveBySources deletes events whose Source is in names.
func (s *Store) RemoveBySources(names ...string) {
	if len(names) == 0 {
		return
	}
	want := make(map[string]struct{}, len(names))
	for _, n := range names {
		want[n] = struct{}{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.order[:0]
	for _, h := range s.order {
		ev := s.byHash[h]
		if ev == nil {
			continue
		}
		if _, ok := want[ev.Source]; ok {
			delete(s.byHash, h)
			continue
		}
		kept = append(kept, h)
	}
	s.order = kept
}

// MatchingHashes returns hashes that match inbox filters.
func (s *Store) MatchingHashes(filter, severity, source, typ string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.order))
	for _, h := range s.order {
		ev := s.byHash[h]
		if ev == nil {
			continue
		}
		if matchEvent(*ev, filter, severity, source, typ) {
			out = append(out, h)
		}
	}
	return out
}

func matchesFilter(ev event.Event, filter string) bool {
	hay := strings.ToLower(strings.Join([]string{
		ev.Type, ev.Message, ev.File, ev.Source, string(ev.Severity),
	}, " "))
	return strings.Contains(hay, filter)
}

func matchEvent(ev event.Event, filter, severity, source, typ string) bool {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter != "" && !matchesFilter(ev, filter) {
		return false
	}
	sev := strings.ToLower(strings.TrimSpace(severity))
	if sev != "" && sev != "all" {
		want := map[string]struct{}{}
		for _, part := range strings.Split(sev, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				want[part] = struct{}{}
			}
		}
		if len(want) > 0 {
			if _, ok := want[string(ev.Severity)]; !ok {
				return false
			}
		}
	}
	if src := strings.TrimSpace(source); src != "" && ev.Source != src {
		return false
	}
	if t := strings.TrimSpace(typ); t != "" && !strings.EqualFold(ev.Type, t) {
		return false
	}
	return true
}
