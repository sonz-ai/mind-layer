package index

import (
	"math"
	"testing"
)

// =============================================================================
// BM25 INDEX TESTS
// =============================================================================

// TestBM25_AddDocument verifies that adding a document increases the doc count
// and makes the document's terms searchable.
func TestBM25_AddDocument(t *testing.T) {
	idx := NewBM25Index()

	idx.AddDocument("doc1", "the quick brown fox")
	idx.AddDocument("doc2", "the lazy brown dog")

	if idx.DocCount() != 2 {
		t.Fatalf("expected DocCount=2, got %d", idx.DocCount())
	}

	// Scoring "fox" should return non-zero for doc1
	score := idx.Score("doc1", "fox")
	if score <= 0 {
		t.Errorf("expected positive score for doc1 on query 'fox', got %.4f", score)
	}

	// doc2 does not contain "fox"
	score2 := idx.Score("doc2", "fox")
	if score2 != 0 {
		t.Errorf("expected zero score for doc2 on query 'fox', got %.4f", score2)
	}
}

// TestBM25_Score_SingleTerm verifies BM25 scoring for a single query term.
func TestBM25_Score_SingleTerm(t *testing.T) {
	idx := NewBM25Index()

	idx.AddDocument("doc1", "pizza is delicious pizza is great")
	idx.AddDocument("doc2", "pasta is wonderful")
	idx.AddDocument("doc3", "pizza rocks")

	// doc1 has "pizza" twice, doc3 has it once, doc2 has it zero times
	score1 := idx.Score("doc1", "pizza")
	score3 := idx.Score("doc3", "pizza")
	score2 := idx.Score("doc2", "pizza")

	if score1 <= 0 {
		t.Errorf("expected positive score for doc1, got %.4f", score1)
	}
	if score3 <= 0 {
		t.Errorf("expected positive score for doc3, got %.4f", score3)
	}
	if score2 != 0 {
		t.Errorf("expected zero score for doc2, got %.4f", score2)
	}

	// doc1 should score higher than doc3 (higher TF for "pizza")
	if score1 <= score3 {
		t.Errorf("doc1 (%.4f) should score higher than doc3 (%.4f) for 'pizza'", score1, score3)
	}
}

// TestBM25_Score_MultipleTerms verifies that BM25 scoring sums across query terms.
func TestBM25_Score_MultipleTerms(t *testing.T) {
	idx := NewBM25Index()

	idx.AddDocument("doc1", "pepperoni pizza with extra cheese")
	idx.AddDocument("doc2", "cheese sandwich with lettuce")
	idx.AddDocument("doc3", "pizza and pasta combo")

	// Query "pizza cheese" should give highest score to doc1 (has both terms)
	score1 := idx.Score("doc1", "pizza cheese")
	score2 := idx.Score("doc2", "pizza cheese")
	score3 := idx.Score("doc3", "pizza cheese")

	if score1 <= 0 {
		t.Fatalf("doc1 should have positive score for 'pizza cheese', got %.4f", score1)
	}

	// doc1 has both terms, so should score highest
	if score1 <= score2 {
		t.Errorf("doc1 (%.4f) should score higher than doc2 (%.4f) for 'pizza cheese'", score1, score2)
	}
	if score1 <= score3 {
		t.Errorf("doc1 (%.4f) should score higher than doc3 (%.4f) for 'pizza cheese'", score1, score3)
	}
}

// TestBM25_IDF_Calculation verifies the inverse document frequency formula:
// IDF = ln((N - n + 0.5) / (n + 0.5) + 1)
// Note: IDF lookups use stemmed terms since Tokenize applies Porter stemming.
func TestBM25_IDF_Calculation(t *testing.T) {
	idx := NewBM25Index()

	idx.AddDocument("doc1", "apple banana cherry")
	idx.AddDocument("doc2", "apple banana date")
	idx.AddDocument("doc3", "apple elderberry fig")

	// Stem the lookup terms to match what the index stores.
	// "apple" -> "appl", "banana" -> "banana", "cherry" -> "cherri"
	stemApple := Stem("apple")
	stemBanana := Stem("banana")
	stemCherry := Stem("cherry")

	// "appl" appears in all 3 docs -> lower IDF
	// "banana" appears in 2 docs -> medium IDF
	// "cherri" appears in 1 doc -> highest IDF
	idfApple := idx.IDF(stemApple)
	idfBanana := idx.IDF(stemBanana)
	idfCherry := idx.IDF(stemCherry)

	if idfApple <= 0 {
		t.Errorf("IDF(%q) should be positive, got %.4f", stemApple, idfApple)
	}
	if idfBanana <= 0 {
		t.Errorf("IDF(%q) should be positive, got %.4f", stemBanana, idfBanana)
	}
	if idfCherry <= 0 {
		t.Errorf("IDF(%q) should be positive, got %.4f", stemCherry, idfCherry)
	}

	// Rarer terms should have higher IDF
	if idfCherry <= idfBanana {
		t.Errorf("IDF(%q)=%.4f should be > IDF(%q)=%.4f", stemCherry, idfCherry, stemBanana, idfBanana)
	}
	if idfBanana <= idfApple {
		t.Errorf("IDF(%q)=%.4f should be > IDF(%q)=%.4f", stemBanana, idfBanana, stemApple, idfApple)
	}

	// Verify numeric value for "cherri" (N=3, n=1):
	// IDF = ln((3 - 1 + 0.5)/(1 + 0.5) + 1) = ln(2.5/1.5 + 1) = ln(2.6667) ~ 0.9808
	expectedCherry := math.Log((3.0-1.0+0.5)/(1.0+0.5) + 1.0)
	if math.Abs(idfCherry-expectedCherry) > 0.01 {
		t.Errorf("IDF(%q) expected %.4f, got %.4f", stemCherry, expectedCherry, idfCherry)
	}
}

// TestBM25_Tokenize verifies that text is split into normalized lowercase terms.
func TestBM25_Tokenize(t *testing.T) {
	idx := NewBM25Index()

	tokens := idx.Tokenize("Hello, World! This is a TEST.")
	if tokens == nil {
		t.Fatal("Tokenize returned nil")
	}
	if len(tokens) == 0 {
		t.Fatal("Tokenize returned empty slice")
	}

	// All tokens should be lowercase
	for _, tok := range tokens {
		for _, r := range tok {
			if r >= 'A' && r <= 'Z' {
				t.Errorf("token %q contains uppercase character", tok)
				break
			}
		}
	}

	// Should contain "hello" and "world" and "test"
	tokenSet := map[string]bool{}
	for _, tok := range tokens {
		tokenSet[tok] = true
	}

	if !tokenSet["hello"] {
		t.Error("expected 'hello' in tokens")
	}
	if !tokenSet["world"] {
		t.Error("expected 'world' in tokens")
	}
	if !tokenSet["test"] {
		t.Error("expected 'test' in tokens")
	}
}

// TestBM25_RemoveDocument verifies that removing a document from the index
// makes it unsearchable and decrements the doc count.
func TestBM25_RemoveDocument(t *testing.T) {
	idx := NewBM25Index()

	idx.AddDocument("doc1", "the quick brown fox")
	idx.AddDocument("doc2", "the lazy brown dog")

	if idx.DocCount() != 2 {
		t.Fatalf("expected DocCount=2 before removal, got %d", idx.DocCount())
	}

	idx.RemoveDocument("doc1")

	if idx.DocCount() != 1 {
		t.Fatalf("expected DocCount=1 after removal, got %d", idx.DocCount())
	}

	// doc1 should no longer be scoreable
	score := idx.Score("doc1", "fox")
	if score != 0 {
		t.Errorf("expected zero score for removed doc1, got %.4f", score)
	}

	// doc2 should still work
	score2 := idx.Score("doc2", "dog")
	if score2 <= 0 {
		t.Errorf("expected positive score for doc2 on 'dog', got %.4f", score2)
	}
}

// TestBM25_Score_NoMatch verifies that querying with terms not present in any
// document returns a score of 0.
func TestBM25_Score_NoMatch(t *testing.T) {
	idx := NewBM25Index()

	idx.AddDocument("doc1", "apple banana cherry")

	score := idx.Score("doc1", "zebra")
	if score != 0 {
		t.Errorf("expected zero score for non-matching query, got %.4f", score)
	}
}

// TestBM25_Score_ExactMatch verifies that a document containing exactly the query
// text scores higher than a document with only partial overlap.
func TestBM25_Score_ExactMatch(t *testing.T) {
	idx := NewBM25Index()

	idx.AddDocument("doc1", "pepperoni pizza")
	idx.AddDocument("doc2", "pepperoni pizza with mushrooms and olives on a thin crust")
	idx.AddDocument("doc3", "cheese pizza")

	// "pepperoni pizza" - doc1 is exact match, doc2 is partial, doc3 only has "pizza"
	score1 := idx.Score("doc1", "pepperoni pizza")
	score2 := idx.Score("doc2", "pepperoni pizza")
	score3 := idx.Score("doc3", "pepperoni pizza")

	if score1 <= 0 {
		t.Fatalf("doc1 should have positive score, got %.4f", score1)
	}

	// Exact match (doc1, shorter doc with both terms) should score >= partial match (doc2, longer doc)
	// BM25 length normalization penalizes longer documents
	if score1 < score2 {
		t.Errorf("exact match doc1 (%.4f) should score >= partial doc2 (%.4f)", score1, score2)
	}

	// doc1 (both terms) should score higher than doc3 (only "pizza")
	if score1 <= score3 {
		t.Errorf("doc1 (%.4f) with both terms should score > doc3 (%.4f) with one term", score1, score3)
	}
}
