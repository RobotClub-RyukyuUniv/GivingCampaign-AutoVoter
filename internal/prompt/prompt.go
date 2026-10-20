// Package prompt は端末での対話入力を扱う。
//
// 標準入力は1つのgoroutineが読み続け、行をチャネルに流す。これによりログイン待ちの間も
// 「URLの貼り付け」と「ブラウザからのコールバック」を同時に待てる。
package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Prompter struct {
	out   io.Writer
	lines chan string
}

func New(in io.Reader, out io.Writer) *Prompter {
	p := &Prompter{out: out, lines: make(chan string)}
	go func() {
		sc := bufio.NewScanner(in)
		sc.Buffer(make([]byte, 1024*1024), 1024*1024)
		for sc.Scan() {
			p.lines <- sc.Text()
		}
		close(p.lines)
	}()
	return p
}

func Default() *Prompter { return New(os.Stdin, os.Stderr) }

// Lines は入力行のチャネル。EOFで閉じられる。
func (p *Prompter) Lines() <-chan string { return p.lines }

func (p *Prompter) Printf(format string, a ...any) { fmt.Fprintf(p.out, format, a...) }

// Line は1行読む。EOFならio.EOFを返す。
func (p *Prompter) Line(label string) (string, error) {
	if label != "" {
		p.Printf("%s", label)
	}
	l, ok := <-p.lines
	if !ok {
		return "", io.EOF
	}
	return strings.TrimSpace(l), nil
}

// Block は /end だけの行か EOF まで複数行を読む（告知文の貼り付け用）。
// 空行は本文として保持し、貼り付けた告知文が途中で切れないようにする。
func (p *Prompter) Block(label string) (string, error) {
	p.Printf("%s\n", label)
	var b strings.Builder
	for {
		l, ok := <-p.lines
		if !ok {
			if b.Len() == 0 {
				return "", io.EOF
			}
			break
		}
		if strings.TrimSpace(l) == "/end" {
			break
		}
		b.WriteString(l)
		b.WriteString("\n")
	}
	return b.String(), nil
}

// String はデフォルト値つきで文字列を尋ねる。
func (p *Prompter) String(label, def string) (string, error) {
	if def != "" {
		label = fmt.Sprintf("%s [%s]", label, def)
	}
	v, err := p.Line(label + ": ")
	if err != nil {
		return "", err
	}
	if v == "" {
		return def, nil
	}
	return v, nil
}

// Confirm は y/n を尋ねる。
func (p *Prompter) Confirm(label string, def bool) (bool, error) {
	hint := "[y/N]"
	if def {
		hint = "[Y/n]"
	}
	for {
		v, err := p.Line(fmt.Sprintf("%s %s: ", label, hint))
		if err != nil {
			return false, err
		}
		switch strings.ToLower(v) {
		case "":
			return def, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}
	}
}

// Option は Select の選択肢。
type Option struct {
	Value string
	Label string
}

// Select は番号で1つ選ばせる。def が選択肢の Value と一致すればそれがデフォルトになる。
func (p *Prompter) Select(label string, opts []Option, def string) (string, error) {
	p.Printf("%s\n", label)
	defIdx := -1
	for i, o := range opts {
		mark := " "
		if o.Value == def {
			mark, defIdx = "*", i
		}
		p.Printf(" %s%2d) %s\n", mark, i+1, o.Label)
	}
	for {
		v, err := p.Line("番号: ")
		if err != nil {
			return "", err
		}
		if v == "" && defIdx >= 0 {
			return opts[defIdx].Value, nil
		}
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= len(opts) {
			return opts[n-1].Value, nil
		}
	}
}

// MultiSelect はカンマ区切りの番号で複数選ばせる。
func (p *Prompter) MultiSelect(label string, opts []Option, defs []string) ([]string, error) {
	p.Printf("%s（カンマ区切りで複数可）\n", label)
	isDef := map[string]bool{}
	for _, d := range defs {
		isDef[d] = true
	}
	for i, o := range opts {
		mark := " "
		if isDef[o.Value] {
			mark = "*"
		}
		p.Printf(" %s%2d) %s\n", mark, i+1, o.Label)
	}
	for {
		v, err := p.Line("番号: ")
		if err != nil {
			return nil, err
		}
		if v == "" && len(defs) > 0 {
			return defs, nil
		}
		var out []string
		ok := true
		for _, s := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == '、' || r == ' ' }) {
			n, err := strconv.Atoi(s)
			if err != nil || n < 1 || n > len(opts) {
				ok = false
				break
			}
			out = append(out, opts[n-1].Value)
		}
		if ok && len(out) > 0 {
			return out, nil
		}
	}
}
