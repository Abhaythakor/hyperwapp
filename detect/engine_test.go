package detect

import (
	"testing"
	"github.com/Abhaythakor/hyperwapp/model"
)

func BenchmarkDetect(b *testing.B) {
	engine, _ := NewWappalyzerEngine()
	headers := map[string][]string{
		"Server": {"Apache"},
		"X-Powered-By": {"PHP/7.4"},
	}
	body := []byte("<html><head><script src='jquery.js'></script></head><body><h1>Hello</h1></body></html>")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Detect(headers, body, model.SourceWappalyzer)
	}
}
