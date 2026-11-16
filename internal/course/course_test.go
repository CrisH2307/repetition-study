package course

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CrisH2307/repetition-study/internal/domain"
	"github.com/CrisH2307/repetition-study/internal/fakes"
)

const u = "user-1"

var ctx = context.Background()

// pdf builds a tiny valid PDF with one page per entry; each page shows its lines.
func pdf(pages ...[]string) []byte {
	var objs []string
	kids := ""
	for i := range pages {
		kids += fmt.Sprintf("%d 0 R ", 4+i*2)
	}
	objs = append(objs, "<< /Type /Catalog /Pages 2 0 R >>",
		fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", kids, len(pages)),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
	for i, lines := range pages {
		stream := "BT /F1 12 Tf 14 TL 50 700 Td "
		for _, l := range lines {
			stream += "(" + l + ") Tj T* "
		}
		stream += "ET"
		objs = append(objs,
			fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents %d 0 R /Resources << /Font << /F1 3 0 R >> >> >>", 5+i*2),
			fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream))
	}
	out := "%PDF-1.4\n"
	var offs []int
	for i, o := range objs {
		offs = append(offs, len(out))
		out += fmt.Sprintf("%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	x := len(out)
	out += fmt.Sprintf("xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, o := range offs {
		out += fmt.Sprintf("%010d 00000 n \n", o)
	}
	out += fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, x)
	return []byte(out)
}

func setup(t *testing.T) (Notes, *fakes.CourseStore, domain.Subject, domain.Concept) {
	st := &fakes.CourseStore{}
	c, _ := st.CreateCourse(ctx, u, domain.Subject{Name: "Calc"})
	k, _ := st.CreateConcept(ctx, u, c.ID, "Limits")
	return Notes{Store: st, Now: func() time.Time { return time.Unix(100, 0) }}, st, c, k
}

func upload(t *testing.T, n Notes, st *fakes.CourseStore, c domain.Subject, file []byte, pasted string) (domain.Upload, []domain.Page) {
	up, err := st.CreateUpload(ctx, u, domain.Upload{SubjectID: c.ID, Filename: "a"}, file, pasted)
	if err != nil {
		t.Fatal(err)
	}
	if err := n.ReadUpload(ctx, u, up.ID); err != nil {
		t.Fatal(err)
	}
	up, _ = st.GetUpload(ctx, u, up.ID)
	ps, _ := st.ListPages(ctx, u, up.ID)
	return up, ps
}

func TestReadPDFPages(t *testing.T) {
	n, st, c, _ := setup(t)
	up, ps := upload(t, n, st, c, pdf([]string{"First page text"}, []string{"Second page text"}), "")
	if up.Status != domain.UploadNeedsChecking || up.Error != "" || len(ps) != 2 {
		t.Fatalf("%+v %d", up, len(ps))
	}
	if !strings.Contains(ps[1].RawText, "Second page text") || ps[1].EditedText != ps[1].RawText {
		t.Fatalf("%q", ps[1].RawText)
	}
	if !ps[0].HasImage {
		t.Fatal("no image")
	}
	if img, err := st.PageImage(ctx, u, ps[0].ID); err != nil || len(img) < 8 || string(img[1:4]) != "PNG" {
		t.Fatal("not a png")
	}
}

func TestReadBadPDFAndPasted(t *testing.T) {
	n, st, c, _ := setup(t)
	up, ps := upload(t, n, st, c, []byte("not a pdf at all"), "")
	if up.Status != domain.UploadNeedsChecking || up.Error == "" || len(ps) != 0 {
		t.Fatalf("%+v", up)
	}
	up, ps = upload(t, n, st, c, nil, "pasted notes")
	if up.Error != "" || len(ps) != 1 || ps[0].RawText != "pasted notes" {
		t.Fatalf("%+v %v", up, ps)
	}
}

func TestReadTooManyPages(t *testing.T) {
	n, st, c, _ := setup(t)
	pages := make([][]string, MaxPages+1)
	for i := range pages {
		pages[i] = []string{"x"}
	}
	up, ps := upload(t, n, st, c, pdf(pages...), "")
	if !strings.Contains(up.Error, "200") || len(ps) != 0 {
		t.Fatalf("%+v %d", up, len(ps))
	}
}

func TestReadyEditAndEmpty(t *testing.T) {
	n, st, c, _ := setup(t)
	_, ps := upload(t, n, st, c, nil, "   ")
	var ue *domain.UserError
	if _, err := n.MarkReady(ctx, u, ps[0].ID); !errors.As(err, &ue) {
		t.Fatalf("empty page: %v", err)
	}
	if _, err := n.EditPage(ctx, u, ps[0].ID, "typed text"); err != nil {
		t.Fatal(err)
	}
	p, err := n.MarkReady(ctx, u, ps[0].ID)
	if err != nil || p.Status != domain.PageReady {
		t.Fatal(p, err)
	}
	bs, _ := st.ReadyBlocks(ctx, u, c.ID)
	if len(bs) != 1 || bs[0].TextNorm != "typed text" || bs[0].VerifiedBy != u || bs[0].VerifiedAt == nil {
		t.Fatalf("%+v", bs)
	}
	if _, err := n.MarkNotReady(ctx, u, p.ID); err != nil {
		t.Fatal(err)
	}
	if bs, _ = st.ReadyBlocks(ctx, u, c.ID); len(bs) != 0 {
		t.Fatal("blocks kept")
	}
}

func TestClaimQuotes(t *testing.T) {
	n, st, c, k := setup(t)
	_, ready := upload(t, n, st, c, pdf([]string{"A limit describes the value a function", "approaches near a point."}), "")
	_, other := upload(t, n, st, c, nil, "A derivative is the slope of the tangent line.")
	n.MarkReady(ctx, u, ready[0].ID)
	bs, _ := st.ReadyBlocks(ctx, u, c.ID)
	in := domain.ClaimInput{Text: "limit", Kind: domain.KindDefinition, Required: true,
		Anchor: domain.Anchor{BlockID: bs[0].ID, Quote: "the value a function approaches near a point"}}
	// quote spans the line break in the PDF
	cl, err := n.AddClaim(ctx, u, k.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	// edit to a bad quote is refused and keeps the old claim
	bad := in
	bad.Anchor.Quote = "A limit is the value a function takes"
	var ue *domain.UserError
	if _, err := n.EditClaim(ctx, u, cl.ID, bad); !errors.As(err, &ue) {
		t.Fatalf("bad edit: %v", err)
	}
	// a page that is not Ready cannot be quoted
	n.MarkReady(ctx, u, other[0].ID)
	bs, _ = st.ReadyBlocks(ctx, u, c.ID)
	n.MarkNotReady(ctx, u, other[0].ID)
	ghost := in
	ghost.Anchor = domain.Anchor{BlockID: bs[len(bs)-1].ID, Quote: "A derivative is the slope of the tangent line."}
	if _, err := n.AddClaim(ctx, u, k.ID, ghost); !errors.As(err, &ue) || !strings.Contains(ue.Msg, "Ready") {
		t.Fatalf("non-ready quote: %v", err)
	}
	// un-ready is refused with a plain reason while the claim anchors to the page
	if _, err := n.MarkNotReady(ctx, u, ready[0].ID); !errors.As(err, &ue) {
		t.Fatalf("unready with claim: %v", err)
	}
	if err := n.DeleteClaim(ctx, u, cl.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := n.MarkNotReady(ctx, u, ready[0].ID); err != nil {
		t.Fatal(err)
	}
	// another user sees nothing
	if _, err := n.AddClaim(ctx, "user-2", k.ID, in); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("other user: %v", err)
	}
}
