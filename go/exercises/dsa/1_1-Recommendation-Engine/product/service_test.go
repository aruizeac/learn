package product_test

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"dsa/1_1-recommendation-engine/product"
)

func TestRecommendationEngine(t *testing.T) {
	// arrange
	productToTest := product.Product{ID: 999, Name: "product-999", Tags: []string{"some-category-1", "some-category-2", "some-category-3"}} // deterministic for tests
	products := make([]product.Product, 1000)
	baseTags := make([]string, 0, 50)
	for i := 0; i < 50; i++ {
		baseTags = append(baseTags, fmt.Sprintf("some-category-%d", i))
	}
	for i := 0; i < 1000; i++ {
		if i == 764 {
			products[i] = productToTest // put item in a specific place to enforce real world scenarios
			continue
		}

		numberOfTags := rand.IntN(5)
		products[i] = product.Product{
			ID:   i,
			Name: fmt.Sprintf("product-%d", i),
		}
		for j := 0; j < numberOfTags; j++ {
			products[i].Tags = append(products[i].Tags, baseTags[rand.IntN(len(baseTags)-1)])
		}
	}
	engine := product.NewRecommendationEngine(products)

	// act
	similarProducts := engine.FindSimilar(productToTest, 3)

	// assert
	t.Logf("similar products: %+v", similarProducts)
	if len(similarProducts) != 3 {
		t.Errorf("expected 3 similar products, got %d", len(similarProducts))
	}
}

func TestFindSimilar_EmptyCatalog(t *testing.T) {
	engine := product.NewRecommendationEngine([]product.Product{})
	query := product.Product{ID: 1, Name: "test", Tags: []string{"tag1"}}

	result := engine.FindSimilar(query, 3)

	if result != nil {
		t.Errorf("expected nil for empty catalog, got %v", result)
	}
}

func TestFindSimilar_SingleProductCatalog(t *testing.T) {
	catalog := []product.Product{
		{ID: 1, Name: "only-product", Tags: []string{"tag1", "tag2"}},
	}
	engine := product.NewRecommendationEngine(catalog)

	// Query with the same product - should return nil since we exclude the query product
	result := engine.FindSimilar(catalog[0], 3)

	if result != nil {
		t.Errorf("expected nil when only product is the query product, got %v", result)
	}
}

func TestFindSimilar_ProductWithNoTags(t *testing.T) {
	catalog := []product.Product{
		{ID: 1, Name: "product-1", Tags: []string{"tag1", "tag2"}},
		{ID: 2, Name: "product-2", Tags: []string{"tag2", "tag3"}},
	}
	engine := product.NewRecommendationEngine(catalog)
	query := product.Product{ID: 99, Name: "no-tags", Tags: []string{}}

	result := engine.FindSimilar(query, 3)

	if result != nil {
		t.Errorf("expected nil for product with no tags, got %v", result)
	}
}

func TestFindSimilar_ProductWithNoMatchingTags(t *testing.T) {
	catalog := []product.Product{
		{ID: 1, Name: "product-1", Tags: []string{"tag1", "tag2"}},
		{ID: 2, Name: "product-2", Tags: []string{"tag2", "tag3"}},
	}
	engine := product.NewRecommendationEngine(catalog)
	query := product.Product{ID: 99, Name: "unique", Tags: []string{"unique-tag-1", "unique-tag-2"}}

	result := engine.FindSimilar(query, 3)

	if result != nil {
		t.Errorf("expected nil for product with no matching tags, got %v", result)
	}
}

func TestFindSimilar_KZero(t *testing.T) {
	catalog := []product.Product{
		{ID: 1, Name: "product-1", Tags: []string{"tag1", "tag2"}},
		{ID: 2, Name: "product-2", Tags: []string{"tag1", "tag3"}},
	}
	engine := product.NewRecommendationEngine(catalog)
	query := product.Product{ID: 99, Name: "query", Tags: []string{"tag1"}}

	result := engine.FindSimilar(query, 0)

	if result != nil {
		t.Errorf("expected nil for k=0, got %v", result)
	}
}

func TestFindSimilar_KGreaterThanAvailable(t *testing.T) {
	catalog := []product.Product{
		{ID: 1, Name: "product-1", Tags: []string{"tag1"}},
		{ID: 2, Name: "product-2", Tags: []string{"tag1"}},
	}
	engine := product.NewRecommendationEngine(catalog)
	query := product.Product{ID: 99, Name: "query", Tags: []string{"tag1"}}

	result := engine.FindSimilar(query, 10)

	if len(result) != 2 {
		t.Errorf("expected 2 similar products (all available), got %d", len(result))
	}
}

func TestFindSimilar_SimilarityRanking(t *testing.T) {
	// Create products with varying overlap with the query
	catalog := []product.Product{
		{ID: 1, Name: "low-similarity", Tags: []string{"tag1"}},                             // 1 overlap
		{ID: 2, Name: "medium-similarity", Tags: []string{"tag1", "tag2"}},                  // 2 overlap
		{ID: 3, Name: "high-similarity", Tags: []string{"tag1", "tag2", "tag3"}},            // 3 overlap
		{ID: 4, Name: "highest-similarity", Tags: []string{"tag1", "tag2", "tag3", "tag4"}}, // 4 overlap
	}
	engine := product.NewRecommendationEngine(catalog)
	query := product.Product{ID: 99, Name: "query", Tags: []string{"tag1", "tag2", "tag3", "tag4"}}

	result := engine.FindSimilar(query, 4)

	if len(result) != 4 {
		t.Fatalf("expected 4 similar products, got %d", len(result))
	}

	// Results should be sorted by similarity (highest first)
	expectedOrder := []int{4, 3, 2, 1}
	for i, p := range result {
		if p.ID != expectedOrder[i] {
			t.Errorf("position %d: expected product ID %d, got %d", i, expectedOrder[i], p.ID)
		}
	}
}

func TestFindSimilar_TopKReturnsOnlyK(t *testing.T) {
	catalog := []product.Product{
		{ID: 1, Name: "p1", Tags: []string{"tag1"}},
		{ID: 2, Name: "p2", Tags: []string{"tag1"}},
		{ID: 3, Name: "p3", Tags: []string{"tag1"}},
		{ID: 4, Name: "p4", Tags: []string{"tag1"}},
		{ID: 5, Name: "p5", Tags: []string{"tag1"}},
	}
	engine := product.NewRecommendationEngine(catalog)
	query := product.Product{ID: 99, Name: "query", Tags: []string{"tag1"}}

	result := engine.FindSimilar(query, 3)

	if len(result) != 3 {
		t.Errorf("expected exactly 3 products, got %d", len(result))
	}
}

func TestFindSimilar_ExcludesQueryProduct(t *testing.T) {
	catalog := []product.Product{
		{ID: 1, Name: "product-1", Tags: []string{"tag1", "tag2"}},
		{ID: 2, Name: "product-2", Tags: []string{"tag1", "tag2", "tag3"}},
		{ID: 3, Name: "product-3", Tags: []string{"tag1"}},
	}
	engine := product.NewRecommendationEngine(catalog)

	// Query with a product that's in the catalog
	result := engine.FindSimilar(catalog[1], 5)

	for _, p := range result {
		if p.ID == catalog[1].ID {
			t.Errorf("query product should not be in results, found ID %d", p.ID)
		}
	}
}

func TestFindSimilar_TieBreakingByProductID(t *testing.T) {
	// Products with same similarity score - tie-breaking by lower product ID
	catalog := []product.Product{
		{ID: 5, Name: "p5", Tags: []string{"tag1"}},
		{ID: 3, Name: "p3", Tags: []string{"tag1"}},
		{ID: 1, Name: "p1", Tags: []string{"tag1"}},
		{ID: 4, Name: "p4", Tags: []string{"tag1"}},
		{ID: 2, Name: "p2", Tags: []string{"tag1"}},
	}
	engine := product.NewRecommendationEngine(catalog)
	query := product.Product{ID: 99, Name: "query", Tags: []string{"tag1"}}

	result := engine.FindSimilar(query, 3)

	if len(result) != 3 {
		t.Fatalf("expected 3 products, got %d", len(result))
	}

	// With tie-breaking by product ID, we should get products with lowest IDs (1, 2, 3)
	foundIDs := make(map[int]bool)
	for _, p := range result {
		foundIDs[p.ID] = true
	}
	for _, expectedID := range []int{1, 2, 3} {
		if !foundIDs[expectedID] {
			t.Errorf("expected to find product ID %d in results (tie-breaking should prefer lower IDs)", expectedID)
		}
	}
}

func TestFindSimilar_MixedSimilarityWithTieBreaking(t *testing.T) {
	catalog := []product.Product{
		{ID: 10, Name: "high-sim-high-id", Tags: []string{"tag1", "tag2", "tag3"}},
		{ID: 1, Name: "high-sim-low-id", Tags: []string{"tag1", "tag2", "tag3"}},
		{ID: 5, Name: "low-sim", Tags: []string{"tag1"}},
	}
	engine := product.NewRecommendationEngine(catalog)
	query := product.Product{ID: 99, Name: "query", Tags: []string{"tag1", "tag2", "tag3"}}

	result := engine.FindSimilar(query, 2)

	if len(result) != 2 {
		t.Fatalf("expected 2 products, got %d", len(result))
	}

	// Both high-similarity products have same score (3), so tie-break by ID
	// Lower ID (1) should come first
	if result[0].ID != 1 {
		t.Errorf("expected first result to be product ID 1 (tie-breaker), got %d", result[0].ID)
	}
	if result[1].ID != 10 {
		t.Errorf("expected second result to be product ID 10, got %d", result[1].ID)
	}
}

func TestFindSimilar_DuplicateTagsInProduct(t *testing.T) {
	catalog := []product.Product{
		{ID: 1, Name: "product-1", Tags: []string{"tag1", "tag2"}},
		{ID: 2, Name: "product-2", Tags: []string{"tag1"}},
	}
	engine := product.NewRecommendationEngine(catalog)
	// Query with duplicate tags - should not double-count
	query := product.Product{ID: 99, Name: "query", Tags: []string{"tag1", "tag1", "tag1"}}

	result := engine.FindSimilar(query, 2)

	if len(result) != 2 {
		t.Fatalf("expected 2 products, got %d", len(result))
	}
}

func TestFindSimilar_LargeCatalog(t *testing.T) {
	// Test with a larger catalog to ensure performance is acceptable
	tags := []string{"electronics", "clothing", "books", "sports", "home", "garden", "toys", "food", "beauty", "auto"}
	catalog := make([]product.Product, 10000)
	for i := 0; i < 10000; i++ {
		numTags := rand.IntN(5) + 1
		productTags := make([]string, 0, numTags)
		for j := 0; j < numTags; j++ {
			productTags = append(productTags, tags[rand.IntN(len(tags))])
		}
		catalog[i] = product.Product{
			ID:   i,
			Name: fmt.Sprintf("product-%d", i),
			Tags: productTags,
		}
	}
	engine := product.NewRecommendationEngine(catalog)
	query := product.Product{ID: 99999, Name: "query", Tags: []string{"electronics", "sports", "home"}}

	result := engine.FindSimilar(query, 10)

	if len(result) > 10 {
		t.Errorf("expected at most 10 products, got %d", len(result))
	}
}

func TestNewRecommendationEngine_BuildsCorrectIndex(t *testing.T) {
	catalog := []product.Product{
		{ID: 1, Name: "p1", Tags: []string{"a", "b"}},
		{ID: 2, Name: "p2", Tags: []string{"b", "c"}},
		{ID: 3, Name: "p3", Tags: []string{"a", "c"}},
	}
	engine := product.NewRecommendationEngine(catalog)

	// Verify by querying - product with tag "a" should find products 1 and 3
	query := product.Product{ID: 99, Name: "query", Tags: []string{"a"}}
	result := engine.FindSimilar(query, 10)

	if len(result) != 2 {
		t.Errorf("expected 2 products with tag 'a', got %d", len(result))
	}

	// Verify IDs found - only products 1 and 3 have tag "a"
	foundIDs := make(map[int]bool)
	for _, p := range result {
		foundIDs[p.ID] = true
	}
	for _, expectedID := range []int{1, 3} {
		if !foundIDs[expectedID] {
			t.Errorf("expected to find product ID %d", expectedID)
		}
	}
}

func TestFindSimilar_MultipleCallsSameEngine(t *testing.T) {
	catalog := []product.Product{
		{ID: 1, Name: "p1", Tags: []string{"tag1", "tag2"}},
		{ID: 2, Name: "p2", Tags: []string{"tag2", "tag3"}},
		{ID: 3, Name: "p3", Tags: []string{"tag1", "tag3"}},
	}
	engine := product.NewRecommendationEngine(catalog)

	// Make multiple queries to ensure engine state is not corrupted
	query1 := product.Product{ID: 99, Name: "q1", Tags: []string{"tag1"}}
	query2 := product.Product{ID: 100, Name: "q2", Tags: []string{"tag2"}}

	result1 := engine.FindSimilar(query1, 3)
	result2 := engine.FindSimilar(query2, 3)

	if len(result1) == 0 || len(result2) == 0 {
		t.Error("expected non-empty results for both queries")
	}
}

func TestFindSimilar_PartialTagOverlap(t *testing.T) {
	catalog := []product.Product{
		{ID: 1, Name: "full-overlap", Tags: []string{"a", "b", "c", "d"}},
		{ID: 2, Name: "partial-overlap", Tags: []string{"a", "b", "x", "y"}},
		{ID: 3, Name: "minimal-overlap", Tags: []string{"a", "x", "y", "z"}},
	}
	engine := product.NewRecommendationEngine(catalog)
	query := product.Product{ID: 99, Name: "query", Tags: []string{"a", "b", "c", "d"}}

	result := engine.FindSimilar(query, 3)

	if len(result) != 3 {
		t.Fatalf("expected 3 products, got %d", len(result))
	}

	// Verify ordering: full overlap (4) > partial (2) > minimal (1)
	if result[0].ID != 1 {
		t.Errorf("expected first result to be product 1 (4 overlaps), got %d", result[0].ID)
	}
	if result[1].ID != 2 {
		t.Errorf("expected second result to be product 2 (2 overlaps), got %d", result[1].ID)
	}
	if result[2].ID != 3 {
		t.Errorf("expected third result to be product 3 (1 overlap), got %d", result[2].ID)
	}
}
