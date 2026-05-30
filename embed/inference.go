package embed

import (
	"encoding/binary"
	"fmt"
	"math"

	ort "github.com/yalue/onnxruntime_go"
)

var ortInitialized bool

func InitONNX() error {
	if ortInitialized {
		return nil
	}
	libPath, err := GetONNXRuntimeLibPath()
	if err != nil {
		return err
	}
	ort.SetSharedLibraryPath(libPath)
	if err := ort.InitializeEnvironment(); err != nil {
		return fmt.Errorf("failed to initialize ONNX environment: %w", err)
	}
	ortInitialized = true
	return nil
}

func GenerateEmbedding(text string) ([]float32, error) {
	tokenIDs, err := Tokenize(text)
	if err != nil {
		return nil, err
	}
	modelPath, err := GetModelPath()
	if err != nil {
		return nil, err
	}

	numTokens := int64(len(tokenIDs))
	inputIDsShape := ort.NewShape(numTokens)
	offsetsShape := ort.NewShape(1)

	inputIDsTensor, err := ort.NewTensor(inputIDsShape, tokenIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to create input_ids tensor: %w", err)
	}
	defer inputIDsTensor.Destroy()

	offsets := []int64{0}
	offsetsTensor, err := ort.NewTensor(offsetsShape, offsets)
	if err != nil {
		return nil, fmt.Errorf("failed to create offsets tensor: %w", err)
	}
	defer offsetsTensor.Destroy()

	outputShape := ort.NewShape(1, int64(EmbeddingDim))
	outputData := make([]byte, outputShape.FlattenedSize()*2)
	outputTensor, err := ort.NewCustomDataTensor(outputShape, outputData, ort.TensorElementDataTypeFloat16)
	if err != nil {
		return nil, fmt.Errorf("failed to create output tensor: %w", err)
	}
	defer outputTensor.Destroy()

	session, err := ort.NewAdvancedSession(
		modelPath,
		[]string{"input_ids", "offsets"},
		[]string{"embeddings"},
		[]ort.Value{inputIDsTensor, offsetsTensor},
		[]ort.Value{outputTensor},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Destroy()

	if err := session.Run(); err != nil {
		return nil, fmt.Errorf("inference failed: %w", err)
	}

	return float16ToFloat32(outputData), nil
}

func float16ToFloat32(data []byte) []float32 {
	result := make([]float32, len(data)/2)
	for i := 0; i < len(result); i++ {
		bits := binary.LittleEndian.Uint16(data[i*2 : i*2+2])
		result[i] = float16BitsToFloat32(bits)
	}
	return result
}

func float16BitsToFloat32(h uint16) float32 {
	sign := uint32((h >> 15) & 0x1)
	exp := uint32((h >> 10) & 0x1f)
	mant := uint32(h & 0x3ff)

	var f uint32
	if exp == 0 {
		if mant == 0 {
			f = sign << 31
		} else {
			exp = 1
			for (mant & 0x400) == 0 {
				mant <<= 1
				exp--
			}
			mant &= 0x3ff
			f = (sign << 31) | ((exp + 127 - 15) << 23) | (mant << 13)
		}
	} else if exp == 31 {
		f = (sign << 31) | (0xff << 23) | (mant << 13)
	} else {
		f = (sign << 31) | ((exp + 127 - 15) << 23) | (mant << 13)
	}

	return math.Float32frombits(f)
}

func CleanupONNX() {
	if ortInitialized {
		ort.DestroyEnvironment()
		ortInitialized = false
	}
}
