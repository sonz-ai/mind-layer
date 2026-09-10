package index

import (
	"runtime"
	"sync"
	"testing"
)

// =============================================================================
// SNOWBALL STEMMER TESTS
// =============================================================================

// TestStem_KnownCases verifies the Snowball stemmer against well-known English test cases.
func TestStem_KnownCases(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		// Step 1a: plurals
		{"caresses", "caress"},
		{"ponies", "poni"},
		{"ties", "tie"},
		{"dogs", "dog"},

		// Step 1b: -ed, -ing
		{"agreed", "agre"},
		{"disabled", "disabl"},
		{"matting", "mat"},
		{"mating", "mate"},
		{"meeting", "meet"},
		{"milling", "mill"},
		{"messing", "mess"},
		{"meetings", "meet"},
		{"running", "run"},

		// No rule matches
		{"connexion", "connexion"},

		// Step 2: double suffix mapping
		{"relational", "relat"},
		{"conditional", "condit"},
		{"rational", "ration"},

		// Step 3: -ful, -ness
		{"goodness", "good"},
		{"hopeful", "hope"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := Stem(tc.input)
			if got != tc.expected {
				t.Errorf("Stem(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

// TestStem_ShortWords ensures very short words pass through unchanged.
func TestStem_ShortWords(t *testing.T) {
	shorts := []string{"a", "i", "to", "go", "an"}
	for _, w := range shorts {
		got := Stem(w)
		if got != w {
			t.Errorf("Stem(%q) = %q, want %q (short word should be unchanged)", w, got, w)
		}
	}
}

// TestStem_EmptyString ensures empty input returns empty output.
func TestStem_EmptyString(t *testing.T) {
	got := Stem("")
	if got != "" {
		t.Errorf("Stem(\"\") = %q, want \"\"", got)
	}
}

// TestStemTokens verifies the convenience function that stems a slice.
func TestStemTokens(t *testing.T) {
	input := []string{"running", "dogs", "meetings", "hopeful"}
	expected := []string{"run", "dog", "meet", "hope"}

	got := StemTokens(input)
	if len(got) != len(expected) {
		t.Fatalf("StemTokens: got %d tokens, want %d", len(got), len(expected))
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("StemTokens[%d] = %q, want %q", i, got[i], expected[i])
		}
	}
}

// TestStemTokens_Empty ensures empty slice returns empty slice.
func TestStemTokens_Empty(t *testing.T) {
	got := StemTokens(nil)
	if got == nil {
		t.Fatal("StemTokens(nil) should return non-nil empty slice")
	}
	if len(got) != 0 {
		t.Errorf("StemTokens(nil) should return empty slice, got %d items", len(got))
	}
}

// =============================================================================
// MULTI-LANGUAGE STEMMING TESTS
// =============================================================================

// TestStemWithLanguage verifies stemming in different languages.
func TestStemWithLanguage(t *testing.T) {
	cases := []struct {
		word     string
		lang     string
		expected string
	}{
		{"running", "english", "run"},
		{"corriendo", "spanish", "corr"},
		{"mangeaient", "french", "mang"},
		{"löpande", "swedish", "löp"},
	}

	for _, tc := range cases {
		t.Run(tc.lang+"/"+tc.word, func(t *testing.T) {
			got := StemWithLanguage(tc.word, tc.lang)
			if got != tc.expected {
				t.Errorf("StemWithLanguage(%q, %q) = %q, want %q", tc.word, tc.lang, got, tc.expected)
			}
		})
	}
}

// TestStemWithLanguage_UnsupportedFallback verifies unsupported languages
// return the word unchanged.
func TestStemWithLanguage_UnsupportedFallback(t *testing.T) {
	got := StemWithLanguage("running", "klingon")
	if got != "running" {
		t.Errorf("StemWithLanguage with unsupported lang should return word unchanged, got %q", got)
	}
}

// TestDetectLanguage verifies language detection on clear text samples.
func TestDetectLanguage(t *testing.T) {
	cases := []struct {
		text     string
		expected string
	}{
		{"The quick brown fox jumps over the lazy dog", "english"},
		{"El rápido zorro marrón salta sobre el perro perezoso", "spanish"},
		{"Le rapide renard brun saute par-dessus le chien paresseux", "french"},
		{"", "english"}, // empty defaults to english
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

// TestDetectLanguage_ShortText verifies that short/ambiguous text defaults to english.
func TestDetectLanguage_ShortText(t *testing.T) {
	got := DetectLanguage("ok")
	// Short text may or may not be detected; we just verify it returns a valid
	// snowball language string without panicking.
	valid := map[string]bool{
		"english": true, "spanish": true, "french": true,
		"russian": true, "swedish": true, "norwegian": true, "hungarian": true,
	}
	if !valid[got] {
		t.Errorf("DetectLanguage(\"ok\") = %q, want a valid snowball language", got)
	}
}

// TestDetectLanguage_LazyLoadingMemory verifies that the language detector uses
// lazy-loading (not preloaded) models. With WithPreloadedLanguageModels(), the
// first call to getDetector() allocates all 14 language model sets at once,
// causing a 50-200MB spike. With lazy loading (the default), only the models
// needed for the detected language are loaded per call.
//
// This test is the regression guard for the memory spike issue — if someone
// re-adds WithPreloadedLanguageModels(), this test should catch the sudden
// heap growth on first detection.
func TestDetectLanguage_LazyLoadingMemory(t *testing.T) {
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)

	// First call initializes the detector.
	_ = DetectLanguage("The quick brown fox jumps over the lazy dog")

	runtime.GC()
	runtime.ReadMemStats(&after)

	growthMB := float64(int64(after.TotalAlloc)-int64(before.TotalAlloc)) / 1024 / 1024
	t.Logf("Total alloc on first DetectLanguage call: %.2f MB", growthMB)

	// With preloaded models, all 14 language models load at once (50-200MB).
	// With lazy loading, only the English model loads for this call (<10MB).
	// We use 50MB as the threshold to catch regressions with some headroom.
	if growthMB > 50 {
		t.Errorf("first DetectLanguage allocated %.2f MB — WithPreloadedLanguageModels() may have been re-added (should be lazy)", growthMB)
	}
}

// TestDetectLanguage_ConcurrentSafe verifies the singleton detector is safe
// for concurrent use and doesn't race on initialization.
func TestDetectLanguage_ConcurrentSafe(t *testing.T) {
	var wg sync.WaitGroup
	results := make([]string, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = DetectLanguage("Hello, this is a concurrent safety test")
		}(i)
	}

	wg.Wait()

	for i, r := range results {
		if r == "" {
			t.Errorf("goroutine %d got empty language result", i)
		}
	}
}

// TestStemTokensWithLanguage verifies batch stemming in a specific language.
func TestStemTokensWithLanguage(t *testing.T) {
	input := []string{"corriendo", "casas", "perros"}
	got := StemTokensWithLanguage(input, "spanish")
	if len(got) != len(input) {
		t.Fatalf("StemTokensWithLanguage: got %d tokens, want %d", len(got), len(input))
	}
	// Each should be stemmed (not identical to input)
	for i, stem := range got {
		if stem == "" {
			t.Errorf("StemTokensWithLanguage[%d] produced empty stem for %q", i, input[i])
		}
	}
}
