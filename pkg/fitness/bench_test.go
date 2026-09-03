package fitness

import (
	"testing"

	"github.com/fow830/ratchet/pkg/tokens"
)

func BenchmarkLayerOf(b *testing.B) {
	cfg := tokens.DefaultConfig("github.com/fow830/ratchet")
	an := NewAnalyzer(cfg)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = an.LayerOf("github.com/fow830/ratchet/internal/domain")
	}
}

func BenchmarkAllowed(b *testing.B) {
	cfg := tokens.DefaultConfig("github.com/fow830/ratchet")
	an := NewAnalyzer(cfg)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = an.Allowed(tokens.LayerDelivery, tokens.LayerDomain)
	}
}
