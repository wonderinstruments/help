package embed

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
	ort "github.com/yalue/onnxruntime_go"
)

const RerankMaxLength = 128

//go:embed models/jina-reranker/model.onnx
var embeddedRerankerModel []byte

//go:embed models/jina-reranker/tokenizer.json
var embeddedRerankerTokenizer []byte

var rerankerTok *tokenizer.Tokenizer

func GetRerankerModelPath() (string, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "jina-reranker.onnx"), nil
}

func getRerankerTokenizerPath() (string, error) {
	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "jina-reranker-tokenizer.json"), nil
}

func EnsureRerankerModel() error {
	modelPath, err := GetRerankerModelPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(modelPath); err == nil {
		return nil
	}
	cacheDir, err := GetCacheDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	if err := os.WriteFile(modelPath, embeddedRerankerModel, 0644); err != nil {
		return fmt.Errorf("failed to write reranker model: %w", err)
	}
	return nil
}

func ensureRerankerTokenizerFile() (string, error) {
	tokPath, err := getRerankerTokenizerPath()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(tokPath); err == nil {
		return tokPath, nil
	}
	cacheDir, err := GetCacheDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}
	if err := os.WriteFile(tokPath, embeddedRerankerTokenizer, 0644); err != nil {
		return "", fmt.Errorf("failed to write reranker tokenizer: %w", err)
	}
	return tokPath, nil
}

func InitRerankerTokenizer() error {
	if rerankerTok != nil {
		return nil
	}
	tokPath, err := ensureRerankerTokenizerFile()
	if err != nil {
		return err
	}
	tok, err := pretrained.FromFile(tokPath)
	if err != nil {
		return fmt.Errorf("failed to load reranker tokenizer from %s: %w", tokPath, err)
	}
	rerankerTok = tok
	return nil
}

func CleanupRerankerTokenizer() {
	rerankerTok = nil
}

type RerankResult struct {
	Index int
	Score float32
}

func Rerank(query string, documents []string) ([]float32, error) {
	if len(documents) == 0 {
		return nil, nil
	}
	if rerankerTok == nil {
		return nil, fmt.Errorf("reranker tokenizer not initialized")
	}

	modelPath, err := GetRerankerModelPath()
	if err != nil {
		return nil, err
	}

	batchSize := len(documents)
	inputIDs := make([]int64, 0, batchSize*RerankMaxLength)
	attentionMask := make([]int64, 0, batchSize*RerankMaxLength)

	for _, doc := range documents {
		ids, mask, err := tokenizeQueryDocPair(query, doc, RerankMaxLength)
		if err != nil {
			return nil, fmt.Errorf("failed to tokenize pair: %w", err)
		}
		inputIDs = append(inputIDs, ids...)
		attentionMask = append(attentionMask, mask...)
	}

	inputShape := ort.NewShape(int64(batchSize), int64(RerankMaxLength))

	inputIDsTensor, err := ort.NewTensor(inputShape, inputIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create input_ids tensor: %w", err)
	}
	defer inputIDsTensor.Destroy()

	attMaskTensor, err := ort.NewTensor(inputShape, attentionMask)
	if err != nil {
		return nil, fmt.Errorf("failed to create attention_mask tensor: %w", err)
	}
	defer attMaskTensor.Destroy()

	outputShape := ort.NewShape(int64(batchSize), 1)
	outputData := make([]float32, batchSize)
	outputTensor, err := ort.NewTensor(outputShape, outputData)
	if err != nil {
		return nil, fmt.Errorf("failed to create output tensor: %w", err)
	}
	defer outputTensor.Destroy()

	session, err := ort.NewAdvancedSession(
		modelPath,
		[]string{"input_ids", "attention_mask"},
		[]string{"logits"},
		[]ort.Value{inputIDsTensor, attMaskTensor},
		[]ort.Value{outputTensor},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create reranker session: %w", err)
	}
	defer session.Destroy()

	if err := session.Run(); err != nil {
		return nil, fmt.Errorf("reranker inference failed: %w", err)
	}

	return outputData, nil
}

func tokenizeQueryDocPair(query, doc string, maxLen int) ([]int64, []int64, error) {
	encoding, err := rerankerTok.EncodePair(query, doc, true)
	if err != nil {
		return nil, nil, err
	}

	ids := encoding.GetIds()
	if len(ids) > maxLen {
		ids = ids[:maxLen]
	}

	attMask := make([]int64, maxLen)
	for i := 0; i < len(ids); i++ {
		attMask[i] = 1
	}

	paddedIDs := make([]int64, maxLen)
	for i, id := range ids {
		paddedIDs[i] = int64(id)
	}
	for i := len(ids); i < maxLen; i++ {
		paddedIDs[i] = 1 // <pad>
	}

	return paddedIDs, attMask, nil
}

func RerankerReady() (bool, error) {
	libPath, err := GetONNXRuntimeLibPath()
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(libPath); os.IsNotExist(err) {
		return false, nil
	}
	modelPath, err := GetRerankerModelPath()
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return false, nil
	}
	return true, nil
}
