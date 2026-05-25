package memory

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ListFilter narrows what List returns. Zero-value = no filter (return
// all non-disabled items, regardless of scope). All non-empty fields
// constrain the result set conjunctively (AND across fields). Within
// Tags the match is any-of (OR within the slice).
type ListFilter struct {
	// Scope matches items via Scope.Matches semantics (empty axis on the
	// filter scope is a wildcard for that axis). Zero-value matches any
	// scope including legacy unscoped items.
	Scope Scope

	// Source / Category are exact matches; empty string = any.
	Source   Source
	Category Category

	// Tags is an any-of filter (case-insensitive). Empty slice = any.
	Tags []string

	// PinnedOnly restricts results to items where IsPinned(item) == true.
	PinnedOnly bool

	// IncludeDisabled controls whether items where IsDisabled(item) ==
	// true appear in the page. Defaults to false (disabled items hidden).
	IncludeDisabled bool

	// MinImportance is an inclusive lower bound on Importance. A value
	// <= 0 means no minimum.
	MinImportance float64
}

// ListPage is one page of items, deterministically ordered by
// (CreatedAt DESC, ID ASC). NextCursor is the empty string when the
// caller has reached the end of the filtered result set; otherwise
// pass it back as the cursor argument to fetch the next page.
type ListPage struct {
	Items      []MemoryItem
	NextCursor string
}

// Lister is implemented by Memory types that support enumeration. It
// is an OPTIONAL interface — the core Memory interface does NOT embed
// Lister, preserving the v0.6 additive-only contract. Callers test for
// the capability with a type assertion:
//
//	if l, ok := mem.(memory.Lister); ok { l.List(...) }
type Lister interface {
	List(ctx context.Context, filter ListFilter, pageSize int, cursor string) (ListPage, error)
}

// --- cursor -----------------------------------------------------------
//
// The cursor is opaque to callers: a base64(json({after_created_at,
// after_id})) blob describing the last item returned on the previous
// page. The next page returns items strictly after that point in the
// (CreatedAt DESC, ID ASC) ordering.

type listCursor struct {
	AfterCreatedAt time.Time `json:"after_created_at"`
	AfterID        string    `json:"after_id"`
}

func encodeCursor(c listCursor) (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func decodeCursor(s string) (listCursor, error) {
	if s == "" {
		return listCursor{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return listCursor{}, fmt.Errorf("memory: bad cursor: %w", err)
	}
	var c listCursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return listCursor{}, fmt.Errorf("memory: bad cursor: %w", err)
	}
	return c, nil
}

// --- filter application -----------------------------------------------

// matchesFilter reports whether it satisfies every constraint in f.
func matchesFilter(it MemoryItem, f ListFilter) bool {
	if !f.IncludeDisabled && IsDisabled(it) {
		return false
	}
	if f.PinnedOnly && !IsPinned(it) {
		return false
	}
	if f.Source != "" && GetSource(it) != f.Source {
		return false
	}
	if f.Category != "" && GetCategory(it) != f.Category {
		return false
	}
	if !f.Scope.IsZero() && !f.Scope.Matches(readScope(it)) {
		return false
	}
	if f.MinImportance > 0 && it.Importance < f.MinImportance {
		return false
	}
	if len(f.Tags) > 0 {
		tagSet := make(map[string]bool, len(it.Tags))
		for _, t := range it.Tags {
			tagSet[strings.ToLower(t)] = true
		}
		match := false
		for _, q := range f.Tags {
			if tagSet[strings.ToLower(q)] {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	return true
}

// listFromStore applies filter + cursor pagination over a scoredStore.
// Sort order is (CreatedAt DESC, ID ASC). pageSize <= 0 defaults to 50.
// Internal helper called by each concrete Memory's List method.
func listFromStore(s *scoredStore, filter ListFilter, pageSize int, cursor string) (ListPage, error) {
	if pageSize <= 0 {
		pageSize = 50
	}
	cur, err := decodeCursor(cursor)
	if err != nil {
		return ListPage{}, err
	}
	items, _ := s.snapshot()
	candidates := make([]MemoryItem, 0, len(items))
	for _, it := range items {
		if !matchesFilter(it, filter) {
			continue
		}
		candidates = append(candidates, it)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if !candidates[i].CreatedAt.Equal(candidates[j].CreatedAt) {
			return candidates[i].CreatedAt.After(candidates[j].CreatedAt)
		}
		return candidates[i].ID < candidates[j].ID
	})
	// skip until strictly past the cursor
	start := 0
	if !cur.AfterCreatedAt.IsZero() || cur.AfterID != "" {
		start = len(candidates)
		for i, it := range candidates {
			if it.CreatedAt.Before(cur.AfterCreatedAt) {
				start = i
				break
			}
			if it.CreatedAt.Equal(cur.AfterCreatedAt) && it.ID > cur.AfterID {
				start = i
				break
			}
		}
	}
	end := start + pageSize
	if end > len(candidates) {
		end = len(candidates)
	}
	page := ListPage{Items: candidates[start:end]}
	if end < len(candidates) {
		last := candidates[end-1]
		nc, err := encodeCursor(listCursor{AfterCreatedAt: last.CreatedAt, AfterID: last.ID})
		if err != nil {
			return ListPage{}, err
		}
		page.NextCursor = nc
	}
	return page, nil
}
