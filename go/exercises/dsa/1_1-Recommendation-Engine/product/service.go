package product

import (
	"container/heap"
	"slices"

	"stl/heapx"
)

// RecommendationEngine should support:
// 1. Building an inverted index from a product catalog
// 2. Finding top-K similar products by overlap score

type RecommendationEngine struct {
	// your fields here
	tagInvertedIndex map[string][]Product
}

// NewRecommendationEngine builds the index from a catalog
func NewRecommendationEngine(catalog []Product) *RecommendationEngine {
	engine := &RecommendationEngine{
		tagInvertedIndex: make(map[string][]Product),
	}
	// This would be performed by the database in a real-world scenario.
	for _, product := range catalog {
		for _, tag := range product.Tags {
			if engine.tagInvertedIndex[tag] == nil {
				engine.tagInvertedIndex[tag] = []Product{product}
				continue
			}
			engine.tagInvertedIndex[tag] = append(engine.tagInvertedIndex[tag], product)
		}
	}
	return engine
}

type countItem struct {
	productID int
	tag       string
	i         int
	count     int
}

// FindSimilar returns the top-k products most similar to the given product
// Similarity = number of overlapping tags
// Exclude the product itself from results
func (e *RecommendationEngine) FindSimilar(product Product, k int) []Product {
	if k <= 0 {
		return nil
	}
	countMap := make(map[int]*countItem) // key = product.ID
	for _, tag := range product.Tags {
		taggedProducts := e.tagInvertedIndex[tag]
		if len(taggedProducts) == 0 {
			continue
		}
		for i, taggedProduct := range taggedProducts {
			if taggedProduct.ID == product.ID {
				continue
			} else if countMap[taggedProduct.ID] != nil {
				countMap[taggedProduct.ID].count++
				continue
			}
			countMap[taggedProduct.ID] = &countItem{productID: taggedProduct.ID, tag: tag, i: i, count: 1}
		}
	}
	similarityHeap := heapx.NewHeap(similarHeapLessFunc, heapx.WithHeapSize(k)) // min-heap
	for _, item := range countMap {
		if similarityHeap.Len() < k {
			heap.Push(similarityHeap, item)
			continue
		}
		// The tie-breaker enforces stable/determinism when two items collide (same count, but different product ID).
		tieBreaker := item.count == similarityHeap.Peek().count && item.productID < similarityHeap.Peek().productID
		if item.count > similarityHeap.Peek().count || tieBreaker {
			heap.Pop(similarityHeap)
			heap.Push(similarityHeap, item)
		}
	}

	if similarityHeap.Len() == 0 {
		return nil
	}

	products := make([]Product, 0, similarityHeap.Len())
	//similarLen := similarityHeap.Len()
	//for i := 0; i < similarLen; i++ {
	//	products = append(products, heap.Pop(similarityHeap).(*SimilarityItem).Product)
	//}
	for item := range similarityHeap.Values() {
		taggedProduct := e.tagInvertedIndex[item.tag][item.i]
		products = append(products, taggedProduct)
	}
	slices.Reverse(products) // min-heap starts from the weakest to the strongest, thus reverse
	return products
}

// - Utils

var similarHeapLessFunc heapx.LessFunc[*countItem] = func(a, b *countItem) bool {
	if a.count != b.count {
		return a.count < b.count
	}
	// Tie-breaker: higher productID is "less" so it gets popped first,
	// keeping lower productIDs in the heap
	return a.productID > b.productID
}
