package index

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"

	"wonderinstruments.com/help/embed"
)

var vecOnce sync.Once

type DB struct {
	conn *sql.DB
}

func Open(path string) (*DB, error) {
	vecOnce.Do(initVec)
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, err
	}
	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS documents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT UNIQUE NOT NULL,
			title TEXT NOT NULL,
			tags TEXT NOT NULL DEFAULT '[]',
			content TEXT NOT NULL,
			content_hash TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			doc_id INTEGER NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
			heading TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			start_line INTEGER NOT NULL,
			end_line INTEGER NOT NULL
		)`,
		fmt.Sprintf(`CREATE VIRTUAL TABLE IF NOT EXISTS vec_chunks USING vec0(
			chunk_id INTEGER PRIMARY KEY,
			embedding FLOAT[%d]
		)`, embed.EmbeddingDim),
		`CREATE VIRTUAL TABLE IF NOT EXISTS fts_chunks USING fts5(
			chunk_id UNINDEXED,
			heading,
			content
		)`,
	}
	for _, stmt := range stmts {
		if _, err := db.conn.Exec(stmt); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, stmt)
		}
	}
	return nil
}

func (db *DB) UpsertDocument(doc *Document) (int64, error) {
	var existingID int64
	var existingHash string
	err := db.conn.QueryRow(
		"SELECT id, content_hash FROM documents WHERE path = ?", doc.Path,
	).Scan(&existingID, &existingHash)

	if err == nil && existingHash == doc.ContentHash {
		return 0, nil // unchanged
	}

	if err == nil {
		db.conn.Exec("DELETE FROM fts_chunks WHERE chunk_id IN (SELECT id FROM chunks WHERE doc_id = ?)", existingID)
		db.conn.Exec("DELETE FROM vec_chunks WHERE chunk_id IN (SELECT id FROM chunks WHERE doc_id = ?)", existingID)
		db.conn.Exec("DELETE FROM chunks WHERE doc_id = ?", existingID)
		db.conn.Exec("DELETE FROM documents WHERE id = ?", existingID)
	}

	tagsJSON := tagsToJSON(doc.Tags)

	res, err := db.conn.Exec(
		"INSERT INTO documents (path, title, tags, content, content_hash) VALUES (?, ?, ?, ?, ?)",
		doc.Path, doc.Title, tagsJSON, doc.Content, doc.ContentHash,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) InsertChunk(docID int64, chunk *Chunk) (int64, error) {
	res, err := db.conn.Exec(
		"INSERT INTO chunks (doc_id, heading, content, start_line, end_line) VALUES (?, ?, ?, ?, ?)",
		docID, chunk.Heading, chunk.Content, chunk.StartLine, chunk.EndLine,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) InsertChunkEmbedding(chunkID int64, embedding []float32) error {
	_, err := db.conn.Exec(
		"INSERT INTO vec_chunks (chunk_id, embedding) VALUES (?, ?)",
		chunkID, serializeEmbedding(embedding),
	)
	return err
}

func (db *DB) InsertChunkFTS(chunkID int64, heading, content string) error {
	_, err := db.conn.Exec(
		"INSERT INTO fts_chunks (chunk_id, heading, content) VALUES (?, ?, ?)",
		chunkID, heading, content,
	)
	return err
}

func (db *DB) SearchChunksFTS(query string, limit int) ([]int64, error) {
	rows, err := db.conn.Query(`
		SELECT chunk_id FROM fts_chunks
		WHERE fts_chunks MATCH ?
		ORDER BY rank
		LIMIT ?
	`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (db *DB) GetChunksByIDs(ids []int64) ([]SearchResult, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	rows, err := db.conn.Query(`
		SELECT c.id, c.doc_id, c.heading, c.content,
		       d.path, d.title, d.tags
		FROM chunks c
		JOIN documents d ON d.id = c.doc_id
		WHERE c.id IN (`+strings.Join(placeholders, ",")+`)
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ChunkID, &r.DocID, &r.Heading, &r.Content, &r.DocPath, &r.DocTitle, &r.Tags); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

type SearchResult struct {
	ChunkID  int64
	Distance float32
	DocID    int64
	DocPath  string
	DocTitle string
	Tags     string
	Heading  string
	Content  string
}

func (db *DB) SearchChunks(queryEmbedding []float32, limit int) ([]SearchResult, error) {
	rows, err := db.conn.Query(`
		SELECT v.chunk_id, v.distance, c.doc_id, c.heading, c.content,
		       d.path, d.title, d.tags
		FROM (
			SELECT chunk_id, distance FROM vec_chunks
			WHERE embedding MATCH ?
			ORDER BY distance LIMIT ?
		) v
		JOIN chunks c ON c.id = v.chunk_id
		JOIN documents d ON d.id = c.doc_id
	`, serializeEmbedding(queryEmbedding), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ChunkID, &r.Distance, &r.DocID, &r.Heading, &r.Content, &r.DocPath, &r.DocTitle, &r.Tags); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *DB) SearchChunksWithTag(queryEmbedding []float32, tag string, limit int) ([]SearchResult, error) {
	// Over-fetch from vec search then filter by tag
	rows, err := db.conn.Query(`
		SELECT v.chunk_id, v.distance, c.doc_id, c.heading, c.content,
		       d.path, d.title, d.tags
		FROM (
			SELECT chunk_id, distance FROM vec_chunks
			WHERE embedding MATCH ?
			ORDER BY distance LIMIT ?
		) v
		JOIN chunks c ON c.id = v.chunk_id
		JOIN documents d ON d.id = c.doc_id
		WHERE d.tags LIKE ?
	`, serializeEmbedding(queryEmbedding), limit*3, `%"`+tag+`"%`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ChunkID, &r.Distance, &r.DocID, &r.Heading, &r.Content, &r.DocPath, &r.DocTitle, &r.Tags); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *DB) ListDocuments(tag string) ([]Document, error) {
	query := "SELECT path, title, tags FROM documents"
	var args []interface{}
	if tag != "" {
		query += ` WHERE tags LIKE ?`
		args = append(args, `%"`+tag+`"%`)
	}
	query += " ORDER BY title"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var d Document
		var tags string
		if err := rows.Scan(&d.Path, &d.Title, &tags); err != nil {
			return nil, err
		}
		d.Tags = parseTags(tags)
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

func (db *DB) ListTags() ([]string, error) {
	rows, err := db.conn.Query("SELECT DISTINCT tags FROM documents")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tagSet := make(map[string]bool)
	for rows.Next() {
		var tagsJSON string
		if err := rows.Scan(&tagsJSON); err != nil {
			return nil, err
		}
		for _, t := range parseTags(tagsJSON) {
			tagSet[t] = true
		}
	}

	var tags []string
	for t := range tagSet {
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func tagsToJSON(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	quoted := make([]string, len(tags))
	for i, t := range tags {
		quoted[i] = `"` + t + `"`
	}
	return "[" + strings.Join(quoted, ",") + "]"
}

func parseTags(tagsJSON string) []string {
	tagsJSON = strings.TrimSpace(tagsJSON)
	if tagsJSON == "[]" || tagsJSON == "" {
		return nil
	}
	tagsJSON = strings.TrimPrefix(tagsJSON, "[")
	tagsJSON = strings.TrimSuffix(tagsJSON, "]")
	parts := strings.Split(tagsJSON, ",")
	var tags []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"`)
		if p != "" {
			tags = append(tags, p)
		}
	}
	return tags
}

func serializeEmbedding(embedding []float32) []byte {
	buf := make([]byte, len(embedding)*4)
	for i, v := range embedding {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}
