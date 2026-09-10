package index

import (
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	sastrawi "github.com/RadhiFadlillah/go-sastrawi"
	"github.com/go-ego/gse"
	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
)

// LanguageTokenizer provides language-aware tokenization for BM25 search.
// Each language backend is lazily initialized on first use to avoid loading
// heavy dictionaries (Chinese ~5MB, Japanese ~15MB) at import time.
type LanguageTokenizer struct {
	mu sync.Mutex

	// Chinese segmenter (gse)
	zhSeg     *gse.Segmenter
	zhInitErr error
	zhOnce    sync.Once

	// Japanese morphological analyzer (kagome)
	jaTok     *tokenizer.Tokenizer
	jaInitErr error
	jaOnce    sync.Once

	// Indonesian stemmer (sastrawi)
	idStemmer *sastrawi.Stemmer
	idOnce    sync.Once
}

// defaultTokenizer is the package-level singleton.
var (
	defaultTokenizer     *LanguageTokenizer
	defaultTokenizerOnce sync.Once
)

func getTokenizer() *LanguageTokenizer {
	defaultTokenizerOnce.Do(func() {
		defaultTokenizer = &LanguageTokenizer{}
	})
	return defaultTokenizer
}

// TokenizeForSearch tokenizes text for BM25 indexing using language-specific
// strategies. The language parameter is a snowball language string (e.g.
// "english", "chinese", "japanese") as returned by DetectLanguage.
func TokenizeForSearch(text, language string) []string {
	return getTokenizer().Tokenize(text, language)
}

// Tokenize dispatches to the appropriate language tokenizer.
func (lt *LanguageTokenizer) Tokenize(text, language string) []string {
	switch language {
	case "chinese":
		return lt.tokenizeChinese(text)
	case "japanese":
		return lt.tokenizeJapanese(text)
	case "korean":
		return lt.tokenizeKorean(text)
	case "indonesian":
		return lt.tokenizeIndonesian(text)
	case "malay", "tagalog":
		return tokenizeWhitespace(text)
	default:
		// Snowball-supported languages: stem each token
		return TokenizeWithLanguage(text, language)
	}
}

// tokenizeChinese uses gse for dictionary-based segmentation with search mode.
func (lt *LanguageTokenizer) tokenizeChinese(text string) []string {
	seg := lt.getChineseSeg()
	if seg == nil {
		return tokenizeWhitespace(text)
	}
	segments := seg.CutSearch(text)
	tokens := make([]string, 0, len(segments))
	for _, s := range segments {
		s = strings.TrimSpace(s)
		if s != "" {
			tokens = append(tokens, strings.ToLower(s))
		}
	}
	return tokens
}

// tokenizeJapanese uses kagome for morphological analysis in Search mode,
// extracting base forms (lemmas) when available. Emits both surface and
// base forms to handle code-switching with other languages.
func (lt *LanguageTokenizer) tokenizeJapanese(text string) []string {
	tok := lt.getJapaneseTokenizer()
	if tok == nil {
		return tokenizeWhitespace(text)
	}
	analyzed := tok.Analyze(text, tokenizer.Search)
	tokens := make([]string, 0, len(analyzed)*2)
	for _, t := range analyzed {
		if t.Class == tokenizer.DUMMY {
			continue
		}
		surface := strings.TrimSpace(strings.ToLower(t.Surface))
		if surface == "" {
			continue
		}
		tokens = append(tokens, surface)
		// Also emit base form if it differs from surface
		if base, ok := t.BaseForm(); ok && base != "" && base != "*" {
			baseLower := strings.ToLower(base)
			if baseLower != surface {
				tokens = append(tokens, baseLower)
			}
		}
	}
	return tokens
}

// tokenizeKorean uses whitespace splitting plus character bigrams for
// Korean text. Korean uses spaces between eojeols (spacing units) but has
// agglutinative morphology. Bigrams provide sub-word matching for BM25.
func (lt *LanguageTokenizer) tokenizeKorean(text string) []string {
	words := strings.Fields(text)
	tokens := make([]string, 0, len(words)*2)
	for _, w := range words {
		lower := strings.ToLower(w)
		// Add the full word
		tokens = append(tokens, lower)
		// Add character bigrams for Korean characters
		bigrams := cjkBigrams(lower)
		tokens = append(tokens, bigrams...)
	}
	return tokens
}

// tokenizeIndonesian uses whitespace splitting + Sastrawi stemmer.
// Emits both the raw token and stemmed form to handle code-switching
// (e.g. mixed Indonesian/English sentences).
func (lt *LanguageTokenizer) tokenizeIndonesian(text string) []string {
	stemmer := lt.getIndonesianStemmer()
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	tokens := make([]string, 0, len(words)*2)
	for _, w := range words {
		lower := strings.ToLower(w)
		if lower == "" {
			continue
		}
		tokens = append(tokens, lower) // Always emit raw form
		if stemmer != nil {
			stemmed := stemmer.Stem(lower)
			if stemmed != lower {
				tokens = append(tokens, stemmed) // Also emit stemmed form
			}
		}
	}
	return tokens
}

// tokenizeWhitespace provides basic whitespace + punctuation splitting
// for languages without specialized tokenizers (Malay, Tagalog).
func tokenizeWhitespace(text string) []string {
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	tokens := make([]string, 0, len(words))
	for _, w := range words {
		lower := strings.ToLower(w)
		if lower != "" {
			tokens = append(tokens, lower)
		}
	}
	return tokens
}

// cjkBigrams returns overlapping character bigrams for CJK characters in the
// input string. Non-CJK characters are skipped. This is the standard approach
// used by Elasticsearch and Bleve for Korean text search.
func cjkBigrams(s string) []string {
	runes := []rune(s)
	var bigrams []string
	var prev rune
	hasPrev := false
	for _, r := range runes {
		if isCJKRune(r) {
			if hasPrev {
				bigrams = append(bigrams, string([]rune{prev, r}))
			}
			prev = r
			hasPrev = true
		} else {
			hasPrev = false
		}
	}
	return bigrams
}

// isCJKRune returns true if the rune is a CJK unified ideograph or Korean
// Hangul syllable/jamo.
func isCJKRune(r rune) bool {
	if utf8.RuneLen(r) < 3 {
		return false
	}
	// CJK Unified Ideographs
	if r >= 0x4E00 && r <= 0x9FFF {
		return true
	}
	// CJK Unified Ideographs Extension A
	if r >= 0x3400 && r <= 0x4DBF {
		return true
	}
	// CJK Unified Ideographs Extension B
	if r >= 0x20000 && r <= 0x2A6DF {
		return true
	}
	// Hangul Syllables
	if r >= 0xAC00 && r <= 0xD7AF {
		return true
	}
	// Hangul Jamo
	if r >= 0x1100 && r <= 0x11FF {
		return true
	}
	// Hangul Compatibility Jamo
	if r >= 0x3130 && r <= 0x318F {
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
	return false
}

// ---------------------------------------------------------------------------
// Lazy initializers for heavy backends
// ---------------------------------------------------------------------------

func (lt *LanguageTokenizer) getChineseSeg() *gse.Segmenter {
	lt.zhOnce.Do(func() {
		seg, err := gse.New()
		if err != nil {
			lt.zhInitErr = err
			return
		}
		lt.zhSeg = &seg
	})
	return lt.zhSeg
}

func (lt *LanguageTokenizer) getJapaneseTokenizer() *tokenizer.Tokenizer {
	lt.jaOnce.Do(func() {
		tok, err := tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())
		if err != nil {
			lt.jaInitErr = err
			return
		}
		lt.jaTok = tok
	})
	return lt.jaTok
}

func (lt *LanguageTokenizer) getIndonesianStemmer() *sastrawi.Stemmer {
	lt.idOnce.Do(func() {
		dict := sastrawi.DefaultDictionary()
		stemmer := sastrawi.NewStemmer(dict)
		lt.idStemmer = &stemmer
	})
	return lt.idStemmer
}
