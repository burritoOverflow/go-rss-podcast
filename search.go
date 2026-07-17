package main

import (
	"github.com/mmcdole/gofeed"
	"github.com/sahilm/fuzzy"
)

// titleSource adapts a slice of feed items to the fuzzy.Source interface so
// episode titles can be fuzzy-matched.
type titleSource []*gofeed.Item

func (t titleSource) String(i int) string { return t[i].Title }
func (t titleSource) Len() int            { return len(t) }

// fuzzyFilter returns the indices of items whose title fuzzy-matches query,
// ordered from best match to worst. An empty query matches every item in
// its original order.
func fuzzyFilter(items []*gofeed.Item, query string) []int {
	if query == "" {
		return allIndices(len(items))
	}

	matches := fuzzy.FindFrom(query, titleSource(items))
	idxs := make([]int, len(matches))
	for i, match := range matches {
		idxs[i] = match.Index
	}
	return idxs
}

// allIndices returns the slice [0, 1, ..., n-1].
func allIndices(n int) []int {
	idxs := make([]int, n)
	for i := range idxs {
		idxs[i] = i
	}
	return idxs
}
