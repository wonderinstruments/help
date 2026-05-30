package index

import (
	"fmt"
	"sort"

	"wonderinstruments.com/help/embed"
)

type RankedResult struct {
	SearchResult
	Score float32
}

func Search(dbPath string, query string, tag string, limit int, rerank bool) ([]RankedResult, error) {
	db, err := Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	if err := embed.InitONNX(); err != nil {
		return nil, err
	}
	if err := embed.InitTokenizer(); err != nil {
		return nil, err
	}

	queryEmb, err := embed.GenerateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	// Semantic search candidates
	semanticLimit := 30
	var semanticResults []SearchResult
	if tag != "" {
		semanticResults, err = db.SearchChunksWithTag(queryEmb, tag, semanticLimit)
	} else {
		semanticResults, err = db.SearchChunks(queryEmb, semanticLimit)
	}
	if err != nil {
		return nil, fmt.Errorf("semantic search: %w", err)
	}

	// BM25 keyword search candidates
	ftsIDs, _ := db.SearchChunksFTS(query, 30)

	// Build ranked lists for RRF
	semanticRanked := make([]int64, len(semanticResults))
	for i, r := range semanticResults {
		semanticRanked[i] = r.ChunkID
	}

	// RRF combine
	combined := rrfCombine([][]int64{semanticRanked, ftsIDs}, 60)

	if len(combined) == 0 {
		return nil, nil
	}

	// Fetch full results for combined IDs
	allResults, err := db.GetChunksByIDs(combined)
	if err != nil {
		return nil, fmt.Errorf("fetch chunks: %w", err)
	}

	// Build lookup for ordering
	resultMap := make(map[int64]SearchResult)
	for _, r := range allResults {
		resultMap[r.ChunkID] = r
	}

	// Order by RRF rank
	ranked := make([]RankedResult, 0, len(combined))
	for i, id := range combined {
		if r, ok := resultMap[id]; ok {
			ranked = append(ranked, RankedResult{
				SearchResult: r,
				Score:        float32(len(combined)-i) / float32(len(combined)), // normalized rank score
			})
		}
	}

	// Rerank with cross-encoder
	if rerank && len(ranked) > 1 {
		if err := embed.EnsureRerankerModel(); err == nil {
			if err := embed.InitRerankerTokenizer(); err == nil {
				docs := make([]string, len(ranked))
				for i, r := range ranked {
					docs[i] = r.Heading + "\n" + r.Content
				}
				scores, err := embed.Rerank(query, docs)
				if err == nil {
					for i := range ranked {
						ranked[i].Score = scores[i]
					}
					sort.Slice(ranked, func(i, j int) bool {
						return ranked[i].Score > ranked[j].Score
					})
				}
			}
		}
	}

	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	return ranked, nil
}

// rrfCombine merges multiple ranked lists using Reciprocal Rank Fusion.
// k is the RRF constant (typically 60).
func rrfCombine(lists [][]int64, k int) []int64 {
	scores := make(map[int64]float64)

	for _, list := range lists {
		for rank, id := range list {
			scores[id] += 1.0 / float64(k+rank+1)
		}
	}

	type scored struct {
		id    int64
		score float64
	}
	var items []scored
	for id, s := range scores {
		items = append(items, scored{id, s})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].score > items[j].score
	})

	result := make([]int64, len(items))
	for i, item := range items {
		result[i] = item.id
	}
	return result
}
