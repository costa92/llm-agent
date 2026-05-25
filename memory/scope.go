package memory

import (
	"context"
)

// Scope identifies a memory partition. Empty fields are wildcards in
// lookup; non-empty fields constrain. Zero-value Scope{} = "match every
// item" (i.e. default behavior pre-v0.7, so existing callers see no
// change).
//
// Scope is propagated through context.Context via WithScope / ScopeFrom
// and read by ScopedManager on every operation.
type Scope struct {
	User    string
	Project string
	Session string
}

// IsZero reports whether the scope matches every item (all fields empty).
func (s Scope) IsZero() bool {
	return s.User == "" && s.Project == "" && s.Session == ""
}

// Equal reports literal field equality.
func (s Scope) Equal(other Scope) bool {
	return s.User == other.User && s.Project == other.Project && s.Session == other.Session
}

// Matches reports whether the (filter) scope s matches the (stored)
// concrete scope. Wildcard rule: an empty field on s ⇒ any value on
// concrete is accepted. If concrete has all-empty fields (legacy /
// unscoped data) and s is non-zero, Matches returns false — legacy
// data is invisible to scoped queries. Callers that want "see legacy
// data" should pass a zero-value Scope (which always returns true).
func (s Scope) Matches(concrete Scope) bool {
	if s.User != "" && s.User != concrete.User {
		return false
	}
	if s.Project != "" && s.Project != concrete.Project {
		return false
	}
	if s.Session != "" && s.Session != concrete.Session {
		return false
	}
	return true
}

// --- ctx propagation ------------------------------------------------------

type scopeCtxKey struct{}

// WithScope returns a child ctx carrying the scope. ScopedManager reads
// it on every operation via ScopeFrom.
func WithScope(ctx context.Context, s Scope) context.Context {
	return context.WithValue(ctx, scopeCtxKey{}, s)
}

// ScopeFrom extracts the scope from ctx. Returns zero-value Scope{}
// (the wildcard) if absent or wrong type.
func ScopeFrom(ctx context.Context) Scope {
	v, ok := ctx.Value(scopeCtxKey{}).(Scope)
	if !ok {
		return Scope{}
	}
	return v
}

// --- metadata serialization ----------------------------------------------
//
// We stamp Scope as a nested map[string]string under metaKeyScope
// ("_scope") so it round-trips cleanly through JSON / generic
// map[string]any consumers without requiring the Scope type itself.

// stampScope writes s into item.Metadata["_scope"]. Initializes Metadata
// if nil. An empty scope (s.IsZero()) does NOT stamp — leaving Metadata
// clean for callers that don't use scoping (backward compat).
func stampScope(item *MemoryItem, s Scope) {
	if s.IsZero() {
		return
	}
	if item.Metadata == nil {
		item.Metadata = make(map[string]any, 1)
	}
	item.Metadata[metaKeyScope] = map[string]string{
		"user":    s.User,
		"project": s.Project,
		"session": s.Session,
	}
}

// readScope extracts the stamped Scope from an item. Returns zero-value
// if absent or malformed. Tolerates both map[string]string (what we
// write) and map[string]any (what JSON round-trips produce).
func readScope(it MemoryItem) Scope {
	if it.Metadata == nil {
		return Scope{}
	}
	raw, ok := it.Metadata[metaKeyScope]
	if !ok {
		return Scope{}
	}
	if m, ok := raw.(map[string]string); ok {
		return Scope{User: m["user"], Project: m["project"], Session: m["session"]}
	}
	if mAny, ok := raw.(map[string]any); ok {
		return Scope{
			User:    stringOr(mAny["user"]),
			Project: stringOr(mAny["project"]),
			Session: stringOr(mAny["session"]),
		}
	}
	return Scope{}
}

func stringOr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
