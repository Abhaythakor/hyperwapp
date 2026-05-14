package fff

import (
	"context"
	"testing"
	"path/filepath"
)

func BenchmarkParseFFF(b *testing.B) {
	root := "../../testdata/fff/multi-domain"
	absRoot, _ := filepath.Abs(root)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch, _ := ParseFFF(context.Background(), absRoot, nil, 1)
		for range ch {
			// Consume
		}
	}
}
