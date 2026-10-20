package prompt

import (
	"io"
	"strings"
	"testing"
)

func TestBlockPreservesBlankLines(t *testing.T) {
	text := "\n\n①最初の団体\nhttps://example.com/first\n\n\n\n \t\n②次の団体\nhttps://example.com/second\n\n\nコメント欄に「応援しています」と記入\n\n\n"
	for _, newline := range []string{"\n", "\r\n"} {
		t.Run(strings.ReplaceAll(newline, "\n", "LF"), func(t *testing.T) {
			input := strings.ReplaceAll(text+" /end \n次の回答\n", "\n", newline)
			p := New(strings.NewReader(input), io.Discard)
			got, err := p.Block("")
			if err != nil || got != text {
				t.Fatalf("Block = %q, %v; want %q", got, err, text)
			}
			answer, err := p.Line("")
			if err != nil || answer != "次の回答" {
				t.Fatalf("Line after Block = %q, %v", answer, err)
			}
		})
	}
}

func TestBlockEOF(t *testing.T) {
	for _, input := range []string{"", "本文\n\n\n続き", "本文に /end が含まれていても終了しない\n"} {
		p := New(strings.NewReader(input), io.Discard)
		got, err := p.Block("")
		if input == "" {
			if got != "" || err != io.EOF {
				t.Fatalf("empty Block = %q, %v; want io.EOF", got, err)
			}
			continue
		}
		want := input
		if !strings.HasSuffix(want, "\n") {
			want += "\n"
		}
		if err != nil || got != want {
			t.Errorf("Block = %q, %v; want %q", got, err, want)
		}
	}
}
