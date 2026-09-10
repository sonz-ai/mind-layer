package index

import (
	"math"
	"strings"
	"unicode"
)

// docEntry stores per-document term frequencies and document length.
type docEntry struct {
	termFreqs map[string]int
	docLen    int
}

// BM25Index provides lightweight lexical search using the BM25 scoring algorithm.
// It maintains an in-memory inverted index of terms to document IDs with term frequencies.
type BM25Index struct {
	// k1 controls term frequency saturation. Typical value: 1.2-2.0
	k1 float64
	// b controls document length normalization. Typical value: 0.75
	b float64

	// docs maps docID -> document entry with term frequencies and length
	docs map[string]*docEntry
	// invertedIndex maps term -> set of docIDs containing that term
	invertedIndex map[string]map[string]bool
	// totalDocLen is the sum of all document lengths (for computing avgdl)
	totalDocLen int
}

// NewBM25Index creates a new BM25Index with default parameters (k1=1.2, b=0.35).
// A lower b value reduces document length normalization, ensuring that higher
// raw term frequency is weighted more heavily than document brevity.
func NewBM25Index() *BM25Index {
	return &BM25Index{
		k1:            1.2,
		b:             0.35,
		docs:          make(map[string]*docEntry),
		invertedIndex: make(map[string]map[string]bool),
	}
}

// AddDocument indexes a document with the given ID and text content.
// The text is tokenized and term frequencies are recorded.
func (idx *BM25Index) AddDocument(docID string, text string) {
	// If already exists, remove first to avoid stale data
	if _, exists := idx.docs[docID]; exists {
		idx.RemoveDocument(docID)
	}

	tokens := idx.Tokenize(text)

	entry := &docEntry{
		termFreqs: make(map[string]int),
		docLen:    len(tokens),
	}

	for _, tok := range tokens {
		entry.termFreqs[tok]++
	}

	idx.docs[docID] = entry
	idx.totalDocLen += entry.docLen

	// Update inverted index
	for term := range entry.termFreqs {
		if idx.invertedIndex[term] == nil {
			idx.invertedIndex[term] = make(map[string]bool)
		}
		idx.invertedIndex[term][docID] = true
	}
}

// RemoveDocument removes a document from the index.
func (idx *BM25Index) RemoveDocument(docID string) {
	entry, exists := idx.docs[docID]
	if !exists {
		return
	}

	// Remove from inverted index
	for term := range entry.termFreqs {
		if docSet, ok := idx.invertedIndex[term]; ok {
			delete(docSet, docID)
			if len(docSet) == 0 {
				delete(idx.invertedIndex, term)
			}
		}
	}

	idx.totalDocLen -= entry.docLen
	delete(idx.docs, docID)
}

// Score computes the BM25 relevance score for a document given a query string.
// Returns 0.0 if the document is not indexed or no query terms match.
func (idx *BM25Index) Score(docID string, query string) float64 {
	entry, exists := idx.docs[docID]
	if !exists {
		return 0.0
	}

	queryTerms := idx.Tokenize(query)
	if len(queryTerms) == 0 {
		return 0.0
	}

	n := len(idx.docs)
	if n == 0 {
		return 0.0
	}
	avgdl := float64(idx.totalDocLen) / float64(n)

	score := 0.0
	dl := float64(entry.docLen)

	for _, term := range queryTerms {
		tf := float64(entry.termFreqs[term])
		if tf == 0 {
			continue
		}

		idf := idx.IDF(term)

		// BM25 term score: IDF * (tf * (k1 + 1)) / (tf + k1 * (1 - b + b * dl / avgdl))
		numerator := tf * (idx.k1 + 1)
		denominator := tf + idx.k1*(1-idx.b+idx.b*dl/avgdl)
		score += idf * numerator / denominator
	}

	return score
}

// ScoreAll returns scores for all indexed documents against the given query.
// Only documents with a non-zero score are included.
func (idx *BM25Index) ScoreAll(query string) map[string]float64 {
	results := make(map[string]float64)
	for docID := range idx.docs {
		s := idx.Score(docID, query)
		if s > 0 {
			results[docID] = s
		}
	}
	return results
}

// Tokenize splits text into normalized, language-aware tokens suitable for
// BM25 indexing. Language is auto-detected from the full text, then the
// appropriate tokenizer is used (Snowball stemming for European languages,
// dictionary segmentation for CJK, Sastrawi for Indonesian, etc.).
func (idx *BM25Index) Tokenize(text string) []string {
	lang := DetectLanguage(text)
	return TokenizeForSearch(text, lang)
}

// TokenizeWithLanguage splits text into normalized lowercase terms stemmed
// in the specified snowball language (e.g. "english", "french", "spanish").
// Emits both the raw token and stemmed form when they differ, so that
// code-switched text (e.g. mixed English/Indonesian) matches regardless
// of which language the query is detected as.
// For non-Snowball languages, use TokenizeForSearch instead.
func TokenizeWithLanguage(text, language string) []string {
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	tokens := make([]string, 0, len(words)*2)
	for _, w := range words {
		lower := strings.ToLower(w)
		if lower == "" {
			continue
		}
		stemmed := StemWithLanguage(lower, language)
		tokens = append(tokens, stemmed)
		if stemmed != lower {
			tokens = append(tokens, lower) // Also emit raw form
		}
	}
	return tokens
}

// DocCount returns the number of documents in the index.
func (idx *BM25Index) DocCount() int {
	return len(idx.docs)
}

// IDF computes the inverse document frequency for a term.
// IDF = ln((N - n + 0.5) / (n + 0.5) + 1) where N is total docs, n is docs containing term.
func (idx *BM25Index) IDF(term string) float64 {
	n := float64(len(idx.docs))
	if n == 0 {
		return 0.0
	}

	df := 0.0
	if docSet, ok := idx.invertedIndex[term]; ok {
		df = float64(len(docSet))
	}

	return math.Log((n-df+0.5)/(df+0.5) + 1.0)
}
