// Package policy — regex source-of-truth for the built-in gates.
//
// This file is the canonical core copy of the PII + injection regex
// tables. The pattern bodies (regex source strings) are lifted verbatim
// from the sister rag repo's guard/redact.go (PII) and guard/inject.go
// (injection) — see the citation block at the bottom. They are COPIED,
// not imported: KC-3 mirrors the otelmodel decorator pattern (per-repo
// source of truth, not a shared upstream); KS-5 keeps the rag repo a
// frozen fixed point (no re-tag, no symbol move).
//
// Q2 ratification — ssn + credit_card omitted from default set
// (US-locale-specific; see policy/doc.go + 36-RESEARCH.md Decision E).
// v1.3-add NewUSLocalePIIRedactor will ship the full rag-parity set.
//
// Both types (piiRule, injectionRule) and both constructors
// (defaultPIIRules, defaultInjectionRules) are unexported — they are
// the package-internal source-of-truth consumed by NewPIIRedactor and
// NewInjectionScanner. Public surface lives in pii.go / injection.go.
package policy

import "regexp"

// piiRule pairs a regex with the placeholder its matches collapse to.
// kind is the human-readable PII category surfaced in audit comments;
// it does NOT appear in the Decision.Reason returned by the gate
// (Reason is "pii_redacted" for the gate; per-rule kind is internal).
type piiRule struct {
	kind        string
	pattern     *regexp.Regexp
	placeholder string
}

// injectionRule is one named injection signature. name surfaces as
// Decision.Reason via InjectionScanner — the four well-known patterns
// (instruction_override / disregard_above / role_override /
// prompt_exfiltration) are stable identifiers users may match on.
type injectionRule struct {
	name    string
	pattern *regexp.Regexp
}

// defaultPIIRules returns the built-in PII rule set for NewPIIRedactor.
//
// Per Q2 ratification (36-RESEARCH.md Decision E + 36-01-PLAN Q1-Q5):
// the core ships exactly 3 language-agnostic patterns — email, phone,
// ipv4. The US-locale entries ssn and credit_card from rag's full set
// are deliberately dropped:
//
//   - ssn (`\d{3}-\d{2}-\d{4}`) false-positives on date-like and ID-
//     like strings in non-US locales.
//   - credit_card (`\d{4}[ -]?\d{4}[ -]?\d{4}[ -]?\d{1,4}`) has a
//     high false-positive rate against any 12-19 digit string and
//     needs a Luhn check to be useful in production.
//
// A v1.3 follow-up will ship NewUSLocalePIIRedactor as the additive
// path for callers who want the full rag-parity set.
//
// Pattern bodies are VERBATIM from the sister rag repo's
// guard/redact.go lines 67-94 (see citation block at end of file).
func defaultPIIRules() []piiRule {
	return []piiRule{
		{
			kind:        "email",
			pattern:     regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`),
			placeholder: "[REDACTED:EMAIL]",
		},
		{
			kind:        "phone",
			pattern:     regexp.MustCompile(`\+?\b\d[\d ()\-]{7,}\d\b`),
			placeholder: "[REDACTED:PHONE]",
		},
		{
			kind:        "ipv4",
			pattern:     regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`),
			placeholder: "[REDACTED:IPV4]",
		},
	}
}

// defaultInjectionRules returns the built-in injection rule set for
// NewInjectionScanner. All 4 patterns from rag's PatternScanner are
// preserved verbatim (no Q2-style drop — these are language-agnostic
// English-prompt signatures and the rag-parity policy applies).
//
// Pattern bodies are VERBATIM from the sister rag repo's
// guard/inject.go lines 49-68 (see citation block at end of file).
// Names preserved exactly so user-facing audit logs
// (BlockedError.Reason) remain stable across the rag/core split.
func defaultInjectionRules() []injectionRule {
	return []injectionRule{
		{
			name:    "instruction_override",
			pattern: regexp.MustCompile(`(?i)ignore\s+(all\s+|the\s+)?(previous|prior|above)\s+(instructions|prompts?)`),
		},
		{
			name:    "disregard_above",
			pattern: regexp.MustCompile(`(?i)disregard\s+(everything\s+|all\s+)?(the\s+)?above`),
		},
		{
			name:    "role_override",
			pattern: regexp.MustCompile(`(?i)(you\s+are\s+now\b|new\s+instructions\s*:|forget\s+(everything|all\s+previous))`),
		},
		{
			name:    "prompt_exfiltration",
			pattern: regexp.MustCompile(`(?i)(reveal|print|show|repeat|display)\s+(your\s+|the\s+)?(system\s+)?(prompt|instructions)`),
		},
	}
}

// --- citation block ---------------------------------------------------
//
// Upstream source files (lifted by copy, NOT imported — KC-3 + KS-5).
// Path is the sister rag repo:
//
//   - guard/redact.go lines 67-94 — PII regex bodies (rag's
//     NewPIIRedactor returns 5 rules; we copy email/phone/ipv4 and
//     drop ssn/credit_card per Q2).
//   - guard/inject.go lines 49-68 — injection regex bodies
//     (rag's NewPatternScanner returns 4 rules; we copy all 4).
//
// A future maintainer comparing this file against the upstream rag
// sources can verify the lift-by-copy at-a-glance: regex source
// strings must match byte-for-byte (subject to the Q2 drop). Names
// (kind / placeholder / rule name) are preserved exactly.
