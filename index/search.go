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

	candidateLimit := limit
	if rerank {
		candidateLimit = 30
	}

	var results []SearchResult
	if tag != "" {
		results, err = db.SearchChunksWithTag(queryEmb, tag, candidateLimit)
	} else {
		results, err = db.SearchChunks(queryEmb, candidateLimit)
	}
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	ranked := make([]RankedResult, len(results))
	for i, r := range results {
		ranked[i] = RankedResult{
			SearchResult: r,
			Score:        1.0 - r.Distance,
		}
	}

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
