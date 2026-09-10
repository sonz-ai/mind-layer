package index

import (
	"sync"

	"github.com/kljensen/snowball"
	lingua "github.com/pemistahl/lingua-go"
)

// linguaToLanguage maps lingua Language constants to our internal language
// strings. These strings are used by both the Snowball stemmer (for European
// languages) and the LanguageTokenizer (for Asian languages).
var linguaToLanguage = map[lingua.Language]string{
	// Snowball-supported languages
	lingua.English:   "english",
	lingua.Spanish:   "spanish",
	lingua.French:    "french",
	lingua.Russian:   "russian",
	lingua.Swedish:   "swedish",
	lingua.Bokmal:    "norwegian",
	lingua.Nynorsk:   "norwegian",
	lingua.Hungarian: "hungarian",

	// Asian languages (tokenized by LanguageTokenizer)
	lingua.Chinese:    "chinese",
	lingua.Japanese:   "japanese",
	lingua.Korean:     "korean",
	lingua.Indonesian: "indonesian",
	lingua.Malay:      "malay",
	lingua.Tagalog:    "tagalog",
}

// detector is a lazily-initialized language detector. Lazy init avoids loading
// n-gram models at import time — the cost is paid once on first use.
var (
	detector     lingua.LanguageDetector
	detectorOnce sync.Once
)

func getDetector() lingua.LanguageDetector {
	detectorOnce.Do(func() {
		languages := make([]lingua.Language, 0, len(linguaToLanguage))
		for lang := range linguaToLanguage {
			languages = append(languages, lang)
		}
		// Default lingua-go behavior is lazy-loading: each language model is loaded
		// on first detection, spreading memory cost over time instead of a single
		// spike when getDetector() is first called. WithPreloadedLanguageModels()
		// was removed because it caused a 50-200MB heap spike on the first request
		// after a fresh deploy (all 14 language models loaded at once).
		detector = lingua.NewLanguageDetectorBuilder().
			FromLanguages(languages...).
			Build()
	})
	return detector
}

// minDetectionLen is the minimum text length (in runes) for reliable language
// detection. Below this threshold we default to English unless the text
// contains non-Latin script characters (CJK, Cyrillic, etc.) which are
// unambiguous even in short strings.
const minDetectionLen = 20

// DetectLanguage returns the internal language string for the given text.
// Falls back to "english" when the language cannot be determined or isn't
// in our supported set.
func DetectLanguage(text string) string {
	if text == "" {
		return "english"
	}

	// For short Latin-script text, language detection is unreliable.
	// Skip detection and default to English unless we see non-Latin chars.
	runes := []rune(text)
	if len(runes) < minDetectionLen && !hasNonLatinScript(runes) {
		return "english"
	}

	lang, ok := getDetector().DetectLanguageOf(text)
	if !ok {
		return "english"
	}
	if name, supported := linguaToLanguage[lang]; supported {
		return name
	}
	return "english"
}

// hasNonLatinScript returns true if any rune in the slice is from a non-Latin
// script (CJK, Cyrillic, Hangul, Arabic, etc.). These scripts are unambiguous
// even in short text.
func hasNonLatinScript(runes []rune) bool {
	for _, r := range runes {
		// CJK Unified Ideographs
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
		// Hiragana
		if r >= 0x3040 && r <= 0x309F {
			return true
		}
		// Katakana
		if r >= 0x30A0 && r <= 0x30FF {
			return true
		}
		// Hangul Syllables
		if r >= 0xAC00 && r <= 0xD7AF {
			return true
		}
		// Cyrillic
		if r >= 0x0400 && r <= 0x04FF {
			return true
		}
		// Arabic
		if r >= 0x0600 && r <= 0x06FF {
			return true
		}
	}
	return false
}

// languageToBCP47 maps internal language names to BCP-47 codes for TTS and
// system prompt language instructions.
var languageToBCP47 = map[string]string{
	"english":    "en-US",
	"spanish":    "es-ES",
	"french":     "fr-FR",
	"russian":    "ru-RU",
	"swedish":    "sv-SE",
	"norwegian":  "nb-NO",
	"hungarian":  "hu-HU",
	"chinese":    "zh-CN",
	"japanese":   "ja-JP",
	"korean":     "ko-KR",
	"indonesian": "id-ID",
	"malay":      "ms-MY",
	"tagalog":    "tl-PH",
}

// DetectLanguageBCP47 returns the BCP-47 language code (e.g. "ja-JP") for the
// given text. Falls back to "en-US" when the language cannot be determined.
func DetectLanguageBCP47(text string) string {
	name := DetectLanguage(text)
	if code, ok := languageToBCP47[name]; ok {
		return code
	}
	return "en-US"
}

// Stem applies the Snowball (Porter2) stemming algorithm to a single lowercase
// word, defaulting to English. For multi-language text, prefer StemWithLanguage
// or StemTokensWithLanguage to avoid per-word detection overhead.
func Stem(word string) string {
	return StemWithLanguage(word, "english")
}

// StemWithLanguage applies the Snowball stemmer for the specified language.
// For languages not supported by Snowball (CJK, Indonesian, Malay, Tagalog),
// the word is returned unchanged.
func StemWithLanguage(word, language string) string {
	if len(word) <= 2 {
		return word
	}
	stemmed, err := snowball.Stem(word, language, true)
	if err != nil {
		// Unsupported language (e.g. CJK) — return unchanged
		return word
	}
	return stemmed
}

// StemTokens applies the English Snowball stemmer to each token.
func StemTokens(tokens []string) []string {
	return StemTokensWithLanguage(tokens, "english")
}

// StemTokensWithLanguage applies the Snowball stemmer for the given language
// to each token in the slice.
func StemTokensWithLanguage(tokens []string, language string) []string {
	result := make([]string, len(tokens))
	for i, t := range tokens {
		result[i] = StemWithLanguage(t, language)
	}
	return result
}
