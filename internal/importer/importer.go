// Package importer reads a CSV of claims into a course (D38). Every claim goes
// through CourseNotes.AddClaim, the same quote check as a typed claim (D28).
package importer

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/CrisH2307/repetition-study/internal/cite"
	"github.com/CrisH2307/repetition-study/internal/domain"
)

// MaxRows is the most data rows one file may have (D38).
const MaxRows = 500

// Importer implements domain.Importer. Courses needs CreateConcept and
// ReadyBlocks; Review needs CourseView (to find concepts by name).
type Importer struct {
	Jobs    domain.ImportStore
	Courses domain.CourseStore
	Review  domain.ReviewStore
	Notes   domain.CourseNotes
	Now     func() time.Time
}

var _ domain.Importer = (*Importer)(nil)

func (im *Importer) now() time.Time {
	if im.Now != nil {
		return im.Now()
	}
	return time.Now().UTC()
}

// safe stops a spreadsheet from running error text as a formula when the
// results are downloaded or pasted into a sheet.
func safe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func fail(rowErrs *[]domain.ImportRowError, row int, msg string) {
	*rowErrs = append(*rowErrs, domain.ImportRowError{Row: row, Error: safe(msg)})
}

func (im *Importer) Import(ctx context.Context, userID, subjectID, filename string, data []byte) (domain.ImportJob, error) {
	j, err := im.Jobs.CreateImport(ctx, userID, subjectID, filename)
	if err != nil {
		return j, err
	}
	j.RowErrors = []domain.ImportRowError{}
	finish := func(st domain.ImportStatus) (domain.ImportJob, error) {
		j.Status = st
		return im.Jobs.FinishImport(ctx, userID, j, im.now())
	}

	r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))))
	r.FieldsPerRecord = -1
	head, err := r.Read()
	if err != nil || !validHeader(head) {
		fail(&j.RowErrors, 1, "The first line must be: concept, claim, kind, quote.")
		return finish(domain.ImportFailed)
	}

	blocks, err := im.Courses.ReadyBlocks(ctx, userID, subjectID)
	if err != nil {
		finish(domain.ImportFailed)
		return j, err
	}
	concepts := map[string]string{} // lower-case name -> id
	if v, err := im.Review.CourseView(ctx, userID, subjectID); err == nil {
		for _, c := range v.Cards {
			concepts[strings.ToLower(c.Name)] = c.ConceptID
		}
	}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		line := 0
		if err == nil {
			line, _ = r.FieldPos(0)
		} else if pe, ok := err.(*csv.ParseError); ok {
			line = pe.StartLine
		}
		if j.RowsTotal == MaxRows {
			fail(&j.RowErrors, line, "Only the first 500 rows are imported. Split the file.")
			break
		}
		j.RowsTotal++
		if err != nil {
			fail(&j.RowErrors, line, "This row could not be read. Check the quotes and commas.")
			continue
		}
		if msg := im.row(ctx, userID, subjectID, rec, blocks, concepts); msg != "" {
			if msg == errInternal {
				finish(domain.ImportFailed)
				return j, errors.New("import stopped")
			}
			fail(&j.RowErrors, line, msg)
			continue
		}
		j.RowsImported++
	}
	return finish(domain.ImportDone)
}

const errInternal = "\x00internal"

func validHeader(h []string) bool {
	want := []string{"concept", "claim", "kind", "quote"}
	if len(h) != 4 {
		return false
	}
	for i, w := range want {
		if strings.ToLower(strings.TrimSpace(h[i])) != w {
			return false
		}
	}
	return true
}

// row returns "" on success, errInternal for a failure that is not the
// student's, else the plain-words reason. It never repeats the cell text.
func (im *Importer) row(ctx context.Context, userID, subjectID string, rec []string, blocks []domain.SourceBlock, concepts map[string]string) string {
	if len(rec) != 4 {
		return "This row needs four columns: concept, claim, kind and quote."
	}
	name, text, kind, quote := strings.TrimSpace(rec[0]), strings.TrimSpace(rec[1]), strings.ToLower(strings.TrimSpace(rec[2])), strings.TrimSpace(rec[3])
	if name == "" || text == "" || kind == "" || quote == "" {
		return "This row needs a concept, a claim, a kind and a quote."
	}
	in := domain.ClaimInput{Text: text, Kind: domain.ClaimKind(kind), Required: true}
	switch in.Kind {
	case domain.KindDefinition, domain.KindStatement, domain.KindCondition, domain.KindMethod:
	default:
		return "The kind must be definition, statement, condition or method."
	}
	for _, b := range blocks {
		if cite.Verify(b.TextMD, quote) {
			in.Anchor = domain.Anchor{BlockID: b.ID, Quote: quote}
			break
		}
	}
	if in.Anchor.BlockID == "" {
		return "That quote is not on a Ready page of this course."
	}
	key := strings.ToLower(name)
	id, ok := concepts[key]
	if !ok {
		c, err := im.Courses.CreateConcept(ctx, userID, subjectID, name)
		if err != nil {
			return userMsg(err)
		}
		id = c.ID
		concepts[key] = id
	}
	if _, err := im.Notes.AddClaim(ctx, userID, id, in); err != nil {
		return userMsg(err)
	}
	return ""
}

func userMsg(err error) string {
	var ue *domain.UserError
	if errors.As(err, &ue) {
		return ue.Msg
	}
	return errInternal
}
