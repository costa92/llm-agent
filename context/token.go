package context

import "unicode"

// TokenCounter estimates the LLM-side token cost of a string.
// Implementations may use a real tokenizer (tiktoken-go) or
// estimate cheaply.
type TokenCounter interface {
	Count(text string) int
}

// SimpleCounter is a no-dep CJK-aware estimator:
//   - ASCII letters / digits: count words → words × 1.3
//   - CJK ideographs: each rune = 1 token
//   - other runes (spaces, punct): contribute to word boundary only
//
// Rough match for OpenAI-style BPE on English (≈1.3 tok/word) and
// for character-level on Chinese (1 tok/CJK char). Intentionally
// pessimistic — better to over-budget than under.
type SimpleCounter struct{}

// Count implements TokenCounter.
func (SimpleCounter) Count(text string) int {
	asciiWords := 0
	cjkChars := 0
	inAsciiWord := false
	for _, r := range text {
		switch {
		case isAsciiAlnum(r):
			if !inAsciiWord {
				asciiWords++
				inAsciiWord = true
			}
		case isCJK(r):
			cjkChars++
			inAsciiWord = false
		default:
			inAsciiWord = false
		}
	}
	// 1.3 tokens per ASCII word + 1 token per CJK char.
	return int(float64(asciiWords)*1.3+0.5) + cjkChars
}

func isAsciiAlnum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// isCJK covers CJK Unified Ideographs + extensions A-G + Hiragana/
// Katakana + Hangul. Good enough for tokenization estimation; not for
// linguistic processing.
func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r)
}
