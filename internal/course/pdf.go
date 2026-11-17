package course

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// MaxPages is the most pages one upload may have.
const MaxPages = 200

var errBadPDF = errors.New("We could not read that PDF. Check that it opens on your computer, or paste the text instead.")

// readPDF splits a PDF into page texts and PNG images (80 dpi) using poppler.
// Course text is untrusted data: it is only stored, never run or sent anywhere.
// An image that fails to render is nil; the page text is still usable.
func readPDF(ctx context.Context, pdf []byte) (texts []string, images [][]byte, err error) {
	dir, err := os.MkdirTemp("", "course-pdf-")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "in.pdf")
	if err := os.WriteFile(file, pdf, 0o600); err != nil {
		return nil, nil, err
	}
	// One extra page tells us the file is over the limit without reading it all.
	out, err := run(ctx, 60*time.Second, "pdftotext", "-layout", "-f", "1", "-l", strconv.Itoa(MaxPages+1), file, "-")
	if err != nil || len(out) == 0 {
		return nil, nil, errBadPDF
	}
	// pdftotext ends every page with a form feed.
	parts := strings.Split(string(out), "\f")
	texts = parts[:len(parts)-1]
	if len(texts) == 0 {
		return nil, nil, errBadPDF
	}
	if len(texts) > MaxPages {
		return nil, nil, errors.New("That PDF has more than 200 pages. Split it into smaller files.")
	}
	for i := range texts {
		n := strconv.Itoa(i + 1)
		prefix := filepath.Join(dir, "p"+n)
		var img []byte
		if _, err := run(ctx, 30*time.Second, "pdftoppm", "-png", "-r", "80", "-f", n, "-l", n, "-singlefile", file, prefix); err == nil {
			img, _ = os.ReadFile(prefix + ".png")
		}
		images = append(images, img)
	}
	return texts, images, nil
}

func run(ctx context.Context, d time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, d)
	defer cancel()
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &out
	err := cmd.Run()
	return out.Bytes(), err
}
