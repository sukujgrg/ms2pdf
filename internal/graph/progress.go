package graph

import (
	"fmt"
	"io"
	"os"
	"time"
)

const progressInterval = 100 * time.Millisecond

type reporter struct {
	out  io.Writer
	tty  bool
	last time.Time
}

func Status(w io.Writer, msg string) {
	newReporter(w).line(msg)
}

func newReporter(w io.Writer) *reporter {
	if w == nil {
		w = os.Stderr
	}
	return &reporter{out: w, tty: isTTY(w)}
}

func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

func (p *reporter) line(text string) {
	if p == nil || !p.tty {
		return
	}
	fmt.Fprintf(p.out, "\r%s\033[K\n", text)
	p.last = time.Time{}
}

func (p *reporter) update(label string, n, total int64, force bool) {
	if p == nil || !p.tty {
		return
	}
	now := time.Now()
	if !force && total > 0 && n < total && now.Sub(p.last) < progressInterval {
		return
	}
	p.last = now
	fmt.Fprintf(p.out, "\r%s\033[K", formatProgress(label, n, total))
}

func (p *reporter) finish() {
	if p == nil || !p.tty {
		return
	}
	fmt.Fprint(p.out, "\n")
	p.last = time.Time{}
}

func (p *reporter) reader(r io.Reader, already, total int64, label string) io.Reader {
	if p == nil || !p.tty {
		return r
	}
	return &countReader{r: r, n: already, total: total, label: label, p: p}
}

func (p *reporter) writer(w io.Writer, limit int64, label string) io.Writer {
	if p == nil || !p.tty {
		if limit <= 0 {
			return w
		}
		return &countWriter{w: w, limit: limit}
	}
	return &countWriter{w: w, limit: limit, label: label, p: p}
}

type countReader struct {
	r        io.Reader
	n, total int64
	label    string
	p        *reporter
}

func (c *countReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	c.p.update(c.label, c.n, c.total, err != nil)
	return n, err
}

type countWriter struct {
	w     io.Writer
	n     int64
	limit int64
	label string
	p     *reporter
}

func (c *countWriter) Write(p []byte) (int, error) {
	if c.limit > 0 && c.n+int64(len(p)) > c.limit {
		return 0, fmt.Errorf("converted PDF exceeds %s", formatBytes(c.limit))
	}
	n, err := c.w.Write(p)
	c.n += int64(n)
	if c.p != nil {
		c.p.update(c.label, c.n, 0, err != nil)
	}
	return n, err
}

func formatProgress(label string, n, total int64) string {
	if total > 0 {
		pct := 0
		if total > 0 {
			pct = int(n * 100 / total)
			if pct > 100 {
				pct = 100
			}
		}
		return fmt.Sprintf("%s  %s / %s  %d%%", label, formatBytes(n), formatBytes(total), pct)
	}
	return fmt.Sprintf("%s  %s", label, formatBytes(n))
}

func formatBytes(n int64) string {
	const (
		kiB = 1024
		miB = 1024 * 1024
	)
	switch {
	case n >= miB:
		return fmt.Sprintf("%.1f MiB", float64(n)/float64(miB))
	case n >= kiB:
		return fmt.Sprintf("%.1f KiB", float64(n)/float64(kiB))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
