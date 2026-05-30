package embed

import (
	"fmt"

	"github.com/CharLemAznable/qwen-tokenizer"
)

var globalTokenizer *tokenizer.Tokenizer

func InitTokenizer() error {
	globalTokenizer = &tokenizer.Tokenizer{}
	return nil
}

func Tokenize(text string) ([]int64, error) {
	if globalTokenizer == nil {
		return nil, fmt.Errorf("tokenizer not initialized")
	}
	ids := globalTokenizer.EncodeOrdinary(text)
	result := make([]int64, len(ids))
	for i, id := range ids {
		result[i] = int64(id)
	}
	return result, nil
}
