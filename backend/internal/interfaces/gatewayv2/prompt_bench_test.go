package gatewayv2

import (
	"fmt"
	"strings"
	"testing"
)

// Bodies shaped like the real provider payloads promptTurn parses on the
// proxy path when conversation recording is on.
func anthropicBody(turns int) []byte {
	var b strings.Builder
	b.WriteString(`{"model":"claude-sonnet-4-5","max_tokens":1024,"messages":[`)
	for i := 0; i < turns; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		role := "user"
		if i%2 == 1 {
			role = "assistant"
		}
		fmt.Fprintf(&b, `{"role":%q,"content":"turn %d: %s"}`, role, i, strings.Repeat("lorem ipsum ", 20))
	}
	b.WriteString(`]}`)
	return []byte(b.String())
}

func BenchmarkPromptTurn(b *testing.B) {
	cases := map[string][]byte{
		"small_1turn":   anthropicBody(1),
		"multi_10turns": anthropicBody(10),
		"large_80turns": anthropicBody(80),
	}
	for name, body := range cases {
		b.Run(fmt.Sprintf("%s_%dB", name, len(body)), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(body)))
			for i := 0; i < b.N; i++ {
				_, _ = promptTurn(body)
			}
		})
	}
}
