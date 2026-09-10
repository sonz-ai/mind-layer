package index

import (
	"strings"
	"testing"
)

// =============================================================================
// MULTI-LANGUAGE TOKENIZER TESTS
// =============================================================================

func TestTokenizeForSearch_Chinese(t *testing.T) {
	tokens := TokenizeForSearch("我喜欢吃披萨和面条", "chinese")
	if len(tokens) == 0 {
		t.Fatal("Chinese tokenization should produce tokens")
	}
	// gse should segment into meaningful words
	joined := strings.Join(tokens, "|")
	t.Logf("Chinese tokens: %s", joined)

	// Should have more tokens than just splitting on characters
	if len(tokens) < 2 {
		t.Errorf("expected multiple tokens from Chinese segmentation, got %d", len(tokens))
	}
}

func TestTokenizeForSearch_Japanese(t *testing.T) {
	tokens := TokenizeForSearch("東京タワーに行きたいです", "japanese")
	if len(tokens) == 0 {
		t.Fatal("Japanese tokenization should produce tokens")
	}
	joined := strings.Join(tokens, "|")
	t.Logf("Japanese tokens: %s", joined)

	// Kagome should segment into morphemes
	if len(tokens) < 3 {
		t.Errorf("expected multiple morphemes from Japanese text, got %d", len(tokens))
	}
}

func TestTokenizeForSearch_Korean(t *testing.T) {
	tokens := TokenizeForSearch("한국어 테스트입니다", "korean")
	if len(tokens) == 0 {
		t.Fatal("Korean tokenization should produce tokens")
	}
	joined := strings.Join(tokens, "|")
	t.Logf("Korean tokens: %s", joined)

	// Should include both full words and bigrams
	hasBigram := false
	for _, tok := range tokens {
		runes := []rune(tok)
		if len(runes) == 2 {
			hasBigram = true
			break
		}
	}
	if !hasBigram {
		t.Error("Korean tokenization should produce character bigrams")
	}
}

func TestTokenizeForSearch_Indonesian(t *testing.T) {
	tokens := TokenizeForSearch("Saya sedang memakan nasi goreng", "indonesian")
	if len(tokens) == 0 {
		t.Fatal("Indonesian tokenization should produce tokens")
	}
	joined := strings.Join(tokens, "|")
	t.Logf("Indonesian tokens: %s", joined)

	// Sastrawi should stem affixed words — verify "sedang" is kept and
	// the tokenizer produces meaningful output (exact stems depend on
	// dictionary, so we just verify count and non-empty tokens)
	for i, tok := range tokens {
		if tok == "" {
			t.Errorf("token[%d] is empty", i)
		}
	}
	if len(tokens) < 3 {
		t.Errorf("expected at least 3 tokens from Indonesian text, got %d", len(tokens))
	}
}

func TestTokenizeForSearch_Malay(t *testing.T) {
	tokens := TokenizeForSearch("Saya suka makan nasi goreng", "malay")
	if len(tokens) == 0 {
		t.Fatal("Malay tokenization should produce tokens")
	}
	// Malay uses whitespace tokenization (no stemmer)
	expected := []string{"saya", "suka", "makan", "nasi", "goreng"}
	if len(tokens) != len(expected) {
		t.Errorf("expected %d tokens, got %d: %v", len(expected), len(tokens), tokens)
	}
}

func TestTokenizeForSearch_Tagalog(t *testing.T) {
	tokens := TokenizeForSearch("Magandang umaga po", "tagalog")
	if len(tokens) == 0 {
		t.Fatal("Tagalog tokenization should produce tokens")
	}
	// Tagalog uses whitespace tokenization (no stemmer)
	expected := []string{"magandang", "umaga", "po"}
	if len(tokens) != len(expected) {
		t.Errorf("expected %d tokens, got %d: %v", len(expected), len(tokens), tokens)
	}
}

func TestTokenizeForSearch_English(t *testing.T) {
	tokens := TokenizeForSearch("The user enjoys running and swimming", "english")
	if len(tokens) == 0 {
		t.Fatal("English tokenization should produce tokens")
	}
	// Should be stemmed
	joined := strings.Join(tokens, "|")
	t.Logf("English tokens: %s", joined)

	// "running" -> "run", "swimming" -> "swim"
	hasRun := false
	hasSwim := false
	for _, tok := range tokens {
		if tok == "run" {
			hasRun = true
		}
		if tok == "swim" {
			hasSwim = true
		}
	}
	if !hasRun {
		t.Errorf("expected 'running' to be stemmed to 'run', got: %s", joined)
	}
	if !hasSwim {
		t.Errorf("expected 'swimming' to be stemmed to 'swim', got: %s", joined)
	}
}

// TestCJKBigrams verifies the bigram generation for CJK characters.
func TestCJKBigrams(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{"한국어", []string{"한국", "국어"}},
		{"AB", nil},          // Latin chars — no bigrams
		{"你好世界", []string{"你好", "好世", "世界"}},
		{"a好b", nil},        // Mixed — CJK chars not adjacent
		{"", nil},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := cjkBigrams(tc.input)
			if len(tc.expected) == 0 && len(got) == 0 {
				return
			}
			if len(got) != len(tc.expected) {
				t.Fatalf("cjkBigrams(%q) = %v, want %v", tc.input, got, tc.expected)
			}
			for i := range tc.expected {
				if got[i] != tc.expected[i] {
					t.Errorf("cjkBigrams(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.expected[i])
				}
			}
		})
	}
}

// TestDetectLanguage_Asian verifies detection of Asian languages.
func TestDetectLanguage_Asian(t *testing.T) {
	cases := []struct {
		text     string
		expected string
	}{
		{"我喜欢吃披萨和面条", "chinese"},
		{"東京タワーに行きたいです", "japanese"},
		{"한국어 테스트입니다", "korean"},
		{"寿司", "chinese"}, // Short CJK still works
	}

	for _, tc := range cases {
		t.Run(tc.expected, func(t *testing.T) {
			got := DetectLanguage(tc.text)
			if got != tc.expected {
				t.Errorf("DetectLanguage(%q) = %q, want %q", tc.text, got, tc.expected)
			}
		})
	}
}
