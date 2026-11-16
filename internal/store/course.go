package store

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/CrisH2307/repetition-study/internal/domain"
)

// Every query joins up to subjects and filters on user_id and archived_at, so
// another user's row looks like a missing row.

const subjectCols = `s.id::text, s.name, s.weekly_hours, s.level, s.exam_date, s.weight`

func scanSubject(r pgx.Row) (x domain.Subject, err error) {
	err = r.Scan(&x.ID, &x.Name, &x.WeeklyHours, &x.Level, &x.ExamDate, &x.Weight)
	if x.ExamDate != nil {
		t := x.ExamDate.UTC()
		x.ExamDate = &t
	}
	return
}

func checkSubject(x domain.Subject) error {
	switch {
	case x.Name == "":
		return &domain.UserError{Msg: "Give the course a name."}
	case x.Level != domain.LevelBeginner && x.Level != domain.LevelIntermediate && x.Level != domain.LevelAdvanced:
		return &domain.UserError{Msg: "Pick beginner, intermediate or advanced."}
	case x.WeeklyHours < 0 || x.Weight < 0:
		return &domain.UserError{Msg: "Hours and weight cannot be negative."}
	}
	return nil
}

func getCourse(ctx context.Context, db q, userID, id string) (domain.Subject, error) {
	x, err := scanSubject(db.QueryRow(ctx, `SELECT `+subjectCols+` FROM subjects s
		WHERE s.id = $1 AND s.user_id = $2 AND s.archived_at IS NULL`, id, userID))
	return x, nf(err)
}

func (s *Store) ListCourses(ctx context.Context, userID string) ([]domain.Subject, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+subjectCols+` FROM subjects s
		WHERE s.user_id = $1 AND s.archived_at IS NULL ORDER BY s.created_at, s.id`, userID)
	return collect(rows, err, scanSubject)
}

func (s *Store) GetCourse(ctx context.Context, userID, id string) (domain.Subject, error) {
	return getCourse(ctx, s.pool, userID, id)
}

func (s *Store) CreateCourse(ctx context.Context, userID string, x domain.Subject) (domain.Subject, error) {
	if err := checkSubject(x); err != nil {
		return domain.Subject{}, err
	}
	var id string
	err := s.pool.QueryRow(ctx, `INSERT INTO subjects (user_id, name, weekly_hours, level, exam_date, weight)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id::text`,
		userID, x.Name, x.WeeklyHours, x.Level, x.ExamDate, x.Weight).Scan(&id)
	if err != nil {
		return domain.Subject{}, err
	}
	return getCourse(ctx, s.pool, userID, id)
}

func (s *Store) UpdateCourse(ctx context.Context, userID string, x domain.Subject) (domain.Subject, error) {
	if err := checkSubject(x); err != nil {
		return domain.Subject{}, err
	}
	if err := affected(s.pool.Exec(ctx, `UPDATE subjects SET name = $3, weekly_hours = $4, level = $5, exam_date = $6, weight = $7
		WHERE id = $1 AND user_id = $2 AND archived_at IS NULL`,
		x.ID, userID, x.Name, x.WeeklyHours, x.Level, x.ExamDate, x.Weight)); err != nil {
		return domain.Subject{}, err
	}
	return getCourse(ctx, s.pool, userID, x.ID)
}

func (s *Store) ArchiveCourse(ctx context.Context, userID, id string) error {
	return affected(s.pool.Exec(ctx, `UPDATE subjects SET archived_at = now()
		WHERE id = $1 AND user_id = $2 AND archived_at IS NULL`, id, userID))
}

func (s *Store) CreateConcept(ctx context.Context, userID, subjectID, name string) (domain.Concept, error) {
	if name == "" {
		return domain.Concept{}, &domain.UserError{Msg: "Give the concept a name."}
	}
	if _, err := getCourse(ctx, s.pool, userID, subjectID); err != nil {
		return domain.Concept{}, err
	}
	c := domain.Concept{SubjectID: subjectID, Name: name, Claims: []domain.Claim{}}
	err := s.pool.QueryRow(ctx, `INSERT INTO concepts (subject_id, name) VALUES ($1, $2) RETURNING id::text`,
		subjectID, name).Scan(&c.ID)
	if pgCode(err) == "23505" {
		return domain.Concept{}, &domain.UserError{Msg: "This course already has a concept with that name."}
	}
	return c, err
}

// conceptSubject returns the course id of a concept the user owns.
func conceptSubject(ctx context.Context, db q, userID, conceptID string) (string, error) {
	var sub string
	err := db.QueryRow(ctx, `SELECT c.subject_id::text FROM concepts c JOIN subjects s ON s.id = c.subject_id
		WHERE c.id = $1 AND s.user_id = $2 AND s.archived_at IS NULL`, conceptID, userID).Scan(&sub)
	return sub, nf(err)
}

func (s *Store) GetConcept(ctx context.Context, userID, id string) (domain.Concept, error) {
	c := domain.Concept{ID: id}
	err := s.pool.QueryRow(ctx, `SELECT c.subject_id::text, c.name FROM concepts c JOIN subjects s ON s.id = c.subject_id
		WHERE c.id = $1 AND s.user_id = $2 AND s.archived_at IS NULL`, id, userID).Scan(&c.SubjectID, &c.Name)
	if err != nil {
		return domain.Concept{}, nf(err)
	}
	c.Claims, err = liveClaims(ctx, s.pool, id)
	return c, err
}

// ---- uploads and pages ----

const uploadSelect = `SELECT u.id::text, u.subject_id::text, u.filename, u.status, u.error, u.source_id::text,
		count(p.id)::int, (count(p.id) FILTER (WHERE p.status = 'ready'))::int, u.created_at
	FROM uploads u JOIN subjects s ON s.id = u.subject_id LEFT JOIN pages p ON p.upload_id = u.id
	WHERE s.user_id = $1 AND s.archived_at IS NULL `

func scanUpload(r pgx.Row) (x domain.Upload, err error) {
	err = r.Scan(&x.ID, &x.SubjectID, &x.Filename, &x.Status, &x.Error, &x.SourceID, &x.PagesTotal, &x.PagesReady, &x.CreatedAt)
	return
}

func getUpload(ctx context.Context, db q, userID, id string) (domain.Upload, error) {
	x, err := scanUpload(db.QueryRow(ctx, uploadSelect+`AND u.id = $2 GROUP BY u.id`, userID, id))
	return x, nf(err)
}

func (s *Store) CreateUpload(ctx context.Context, userID string, u domain.Upload, pdf []byte, pasted string) (domain.Upload, error) {
	switch {
	case (len(pdf) == 0) == (pasted == ""):
		return domain.Upload{}, &domain.UserError{Msg: "Add a PDF or paste some text, not both."}
	case len(pdf) > 20<<20:
		return domain.Upload{}, &domain.UserError{Msg: "The file is larger than 20 MB."}
	}
	if _, err := getCourse(ctx, s.pool, userID, u.SubjectID); err != nil {
		return domain.Upload{}, err
	}
	var p *string
	if pasted != "" {
		p = &pasted
	}
	if len(pdf) == 0 {
		pdf = nil
	}
	var id string
	if err := s.pool.QueryRow(ctx, `INSERT INTO uploads (subject_id, filename, pdf, pasted_text) VALUES ($1, $2, $3, $4) RETURNING id::text`,
		u.SubjectID, u.Filename, pdf, p).Scan(&id); err != nil {
		return domain.Upload{}, err
	}
	return getUpload(ctx, s.pool, userID, id)
}

func (s *Store) GetUpload(ctx context.Context, userID, id string) (domain.Upload, error) {
	return getUpload(ctx, s.pool, userID, id)
}

func (s *Store) ListUploads(ctx context.Context, userID, subjectID string) ([]domain.Upload, error) {
	if _, err := getCourse(ctx, s.pool, userID, subjectID); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, uploadSelect+`AND u.subject_id = $2 GROUP BY u.id ORDER BY u.created_at, u.id`, userID, subjectID)
	return collect(rows, err, scanUpload)
}

func (s *Store) UploadFile(ctx context.Context, userID, id string) ([]byte, string, error) {
	var pdf []byte
	var pasted *string
	err := s.pool.QueryRow(ctx, `SELECT u.pdf, u.pasted_text FROM uploads u JOIN subjects s ON s.id = u.subject_id
		WHERE u.id = $1 AND s.user_id = $2 AND s.archived_at IS NULL`, id, userID).Scan(&pdf, &pasted)
	if err != nil {
		return nil, "", nf(err)
	}
	if pasted == nil {
		return pdf, "", nil
	}
	return pdf, *pasted, nil
}

func (s *Store) SetUploadStatus(ctx context.Context, userID, id string, st domain.UploadStatus, msg string) error {
	if st != domain.UploadReading && st != domain.UploadNeedsChecking && st != domain.UploadReady {
		return &domain.UserError{Msg: "Unknown upload status."}
	}
	return affected(s.pool.Exec(ctx, `UPDATE uploads u SET status = $3, error = $4 FROM subjects s
		WHERE u.id = $1 AND s.id = u.subject_id AND s.user_id = $2 AND s.archived_at IS NULL`, id, userID, st, msg))
}

// SavePages writes the pages in one transaction. A page number that exists and
// is still unchecked is overwritten (a re-read); a checked one is kept.
func (s *Store) SavePages(ctx context.Context, userID, uploadID string, pages []domain.Page, images [][]byte) error {
	if _, err := getUpload(ctx, s.pool, userID, uploadID); err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		for i, p := range pages {
			var img []byte
			if i < len(images) {
				img = images[i]
			}
			if _, err := tx.Exec(ctx, `INSERT INTO pages (upload_id, number, raw_text, edited_text, image_png) VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (upload_id, number) DO UPDATE SET raw_text = $3, edited_text = $4, image_png = $5, updated_at = now()
				WHERE pages.status = 'needs_checking'`, uploadID, p.Number, p.RawText, p.EditedText, img); err != nil {
				return err
			}
		}
		return nil
	})
}

const pageSelect = `SELECT p.id::text, p.upload_id::text, p.number, p.raw_text, p.edited_text, p.status,
		p.image_png IS NOT NULL, p.updated_at
	FROM pages p JOIN uploads u ON u.id = p.upload_id JOIN subjects s ON s.id = u.subject_id
	WHERE s.user_id = $1 AND s.archived_at IS NULL `

func scanPage(r pgx.Row) (x domain.Page, err error) {
	err = r.Scan(&x.ID, &x.UploadID, &x.Number, &x.RawText, &x.EditedText, &x.Status, &x.HasImage, &x.UpdatedAt)
	return
}

func getPage(ctx context.Context, db q, userID, id string) (domain.Page, error) {
	x, err := scanPage(db.QueryRow(ctx, pageSelect+`AND p.id = $2`, userID, id))
	return x, nf(err)
}

func (s *Store) ListPages(ctx context.Context, userID, uploadID string) ([]domain.Page, error) {
	if _, err := getUpload(ctx, s.pool, userID, uploadID); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, pageSelect+`AND p.upload_id = $2 ORDER BY p.number`, userID, uploadID)
	return collect(rows, err, scanPage)
}

func (s *Store) GetPage(ctx context.Context, userID, id string) (domain.Page, error) {
	return getPage(ctx, s.pool, userID, id)
}

func (s *Store) PageImage(ctx context.Context, userID, id string) ([]byte, error) {
	var img []byte
	err := s.pool.QueryRow(ctx, `SELECT p.image_png FROM pages p JOIN uploads u ON u.id = p.upload_id JOIN subjects s ON s.id = u.subject_id
		WHERE p.id = $1 AND s.user_id = $2 AND s.archived_at IS NULL`, id, userID).Scan(&img)
	if err = nf(err); err == nil && img == nil {
		err = domain.ErrNotFound
	}
	return img, err
}

func (s *Store) EditPage(ctx context.Context, userID, id, text string) (domain.Page, error) {
	if err := affected(s.pool.Exec(ctx, `UPDATE pages p SET edited_text = $3, updated_at = now()
		FROM uploads u JOIN subjects s ON s.id = u.subject_id
		WHERE p.id = $1 AND u.id = p.upload_id AND s.user_id = $2 AND s.archived_at IS NULL`, id, userID, text)); err != nil {
		return domain.Page{}, err
	}
	return getPage(ctx, s.pool, userID, id)
}

// Blocks of a page get ordinal number*pageOrdinalStride+i, so pages of one
// upload never collide on UNIQUE (source_id, ordinal) and sort in page order.
const pageOrdinalStride = 100000

func (s *Store) SetPageReady(ctx context.Context, userID, id string, blocks []domain.SourceBlock) (domain.Page, error) {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var uploadID, subjectID, filename string
		var sourceID *string
		var number int
		if err := tx.QueryRow(ctx, `SELECT p.upload_id::text, p.number, u.subject_id::text, u.source_id::text, u.filename
			FROM pages p JOIN uploads u ON u.id = p.upload_id JOIN subjects s ON s.id = u.subject_id
			WHERE p.id = $1 AND s.user_id = $2 AND s.archived_at IS NULL FOR UPDATE OF p, u`, id, userID).
			Scan(&uploadID, &number, &subjectID, &sourceID, &filename); err != nil {
			return nf(err)
		}
		if sourceID == nil {
			var sid string
			if err := tx.QueryRow(ctx, `INSERT INTO sources (kind, title, text, subject_id, reviewed_by, reviewed_at)
				VALUES ('course', $1, '', $2, $3, now()) RETURNING id::text`, filename, subjectID, userID).Scan(&sid); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE uploads SET source_id = $2 WHERE id = $1`, uploadID, sid); err != nil {
				return err
			}
			sourceID = &sid
		}
		if err := dropPageBlocks(ctx, tx, id); err != nil {
			return err
		}
		for i, b := range blocks {
			pg := number
			if b.Page != nil {
				pg = *b.Page
			}
			by := b.VerifiedBy
			if by == "" {
				by = userID
			}
			if _, err := tx.Exec(ctx, `INSERT INTO source_blocks (source_id, page, ordinal, kind, text_md, text_norm, verified_by, verified_at, page_id)
				VALUES ($1, $2, $3, $4, $5, $6, $7, now(), $8)`,
				*sourceID, pg, number*pageOrdinalStride+i, b.Kind, b.TextMD, b.TextNorm, by, id); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE pages SET status = 'ready', updated_at = now() WHERE id = $1`, id); err != nil {
			return err
		}
		return refreshUpload(ctx, tx, uploadID, *sourceID)
	})
	if err != nil {
		return domain.Page{}, err
	}
	return getPage(ctx, s.pool, userID, id)
}

func (s *Store) SetPageNotReady(ctx context.Context, userID, id string) (domain.Page, error) {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var uploadID string
		var sourceID *string
		if err := tx.QueryRow(ctx, `SELECT p.upload_id::text, u.source_id::text
			FROM pages p JOIN uploads u ON u.id = p.upload_id JOIN subjects s ON s.id = u.subject_id
			WHERE p.id = $1 AND s.user_id = $2 AND s.archived_at IS NULL FOR UPDATE OF p, u`, id, userID).
			Scan(&uploadID, &sourceID); err != nil {
			return nf(err)
		}
		if err := dropPageBlocks(ctx, tx, id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE pages SET status = 'needs_checking', updated_at = now() WHERE id = $1`, id); err != nil {
			return err
		}
		if sourceID == nil {
			return nil
		}
		return refreshUpload(ctx, tx, uploadID, *sourceID)
	})
	if err != nil {
		return domain.Page{}, err
	}
	return getPage(ctx, s.pool, userID, id)
}

// dropPageBlocks deletes the page's blocks. A live claim anchored to one
// refuses. Deleted claims keep their text but lose the anchor (their row must
// stay, labels and cards point at it).
func dropPageBlocks(ctx context.Context, tx pgx.Tx, pageID string) error {
	var n int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM claims c JOIN source_blocks b ON b.id = c.anchor_block_id
		WHERE b.page_id = $1 AND c.deleted_at IS NULL`, pageID).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return &domain.UserError{Msg: "A claim quotes this page. Remove that claim first."}
	}
	if _, err := tx.Exec(ctx, `UPDATE claims SET anchor_block_id = NULL, anchor_quote = NULL
		WHERE anchor_block_id IN (SELECT id FROM source_blocks WHERE page_id = $1)`, pageID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `DELETE FROM source_blocks WHERE page_id = $1`, pageID)
	return err
}

// refreshUpload rebuilds the course source's text from its page blocks and
// moves the upload between needs_checking and ready as its pages change.
func refreshUpload(ctx context.Context, tx pgx.Tx, uploadID, sourceID string) error {
	if _, err := tx.Exec(ctx, `UPDATE sources SET text = coalesce((
			SELECT string_agg(b.text_md, E'\n\n' ORDER BY b.ordinal) FROM source_blocks b WHERE b.source_id = $1), '')
		WHERE id = $1`, sourceID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE uploads SET status = CASE
			WHEN EXISTS (SELECT 1 FROM pages WHERE upload_id = $1) AND NOT EXISTS (SELECT 1 FROM pages WHERE upload_id = $1 AND status <> 'ready') THEN 'ready'
			WHEN status = 'ready' THEN 'needs_checking' ELSE status END
		WHERE id = $1 AND status <> 'reading'`, uploadID)
	return err
}

func (s *Store) ReadyBlocks(ctx context.Context, userID, subjectID string) ([]domain.SourceBlock, error) {
	if _, err := getCourse(ctx, s.pool, userID, subjectID); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT b.id::text, b.source_id::text, b.page, b.ordinal, b.kind, b.text_md, b.text_norm, b.verified_by, b.verified_at
		FROM source_blocks b JOIN pages p ON p.id = b.page_id JOIN uploads u ON u.id = p.upload_id
		WHERE u.subject_id = $1 AND p.status = 'ready'
		ORDER BY u.created_at, u.id, p.number, b.ordinal`, subjectID)
	return collect(rows, err, func(r pgx.Row) (b domain.SourceBlock, err error) {
		err = r.Scan(&b.ID, &b.SourceID, &b.Page, &b.Ordinal, &b.Kind, &b.TextMD, &b.TextNorm, &b.VerifiedBy, &b.VerifiedAt)
		return
	})
}

// CourseSources returns the course's approved sources that have text: Phase 0
// sources and upload sources with at least one Ready page.
func (s *Store) CourseSources(ctx context.Context, userID, subjectID string) ([]domain.Source, error) {
	if _, err := getCourse(ctx, s.pool, userID, subjectID); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `SELECT id::text, kind, title, url, publisher, license, reviewed_by, reviewed_at, text
		FROM sources WHERE subject_id = $1 AND kind = 'course' AND text <> '' ORDER BY created_at, id`, subjectID)
	return collect(rows, err, func(r pgx.Row) (x domain.Source, err error) {
		err = r.Scan(&x.ID, &x.Kind, &x.Title, &x.URL, &x.Publisher, &x.License, &x.ReviewedBy, &x.ReviewedAt, &x.Text)
		return
	})
}

// ---- claims ----

const claimCols = `c.id::text, c.concept_id::text, c.text, c.source_id::text, c.locator, c.required,
	c.kind, c.author_user_id::text, c.anchor_block_id::text, c.anchor_quote`

func scanClaim(r pgx.Row) (x domain.Claim, err error) {
	var block, quote *string
	if err = r.Scan(&x.ID, &x.ConceptID, &x.Text, &x.SourceID, &x.Locator, &x.Required, &x.Kind, &x.AuthorUserID, &block, &quote); err != nil {
		return
	}
	if block != nil && quote != nil {
		x.Anchor = &domain.Anchor{BlockID: *block, Quote: *quote}
	}
	return
}

func liveClaims(ctx context.Context, db q, conceptID string) ([]domain.Claim, error) {
	rows, err := db.Query(ctx, `SELECT `+claimCols+` FROM claims c WHERE c.concept_id = $1 AND c.deleted_at IS NULL
		ORDER BY c.created_at, c.id`, conceptID)
	return collect(rows, err, scanClaim)
}

func (s *Store) ListClaims(ctx context.Context, userID, conceptID string) ([]domain.Claim, error) {
	if _, err := conceptSubject(ctx, s.pool, userID, conceptID); err != nil {
		return nil, err
	}
	return liveClaims(ctx, s.pool, conceptID)
}

var claimKinds = map[domain.ClaimKind]bool{domain.KindDefinition: true, domain.KindStatement: true, domain.KindCondition: true, domain.KindMethod: true}

// anchorSource checks the block is on a Ready page of the course and returns
// its source and a locator. The quote itself is internal/cite's job.
func anchorSource(ctx context.Context, db q, subjectID string, in domain.ClaimInput) (sourceID, locator string, err error) {
	if in.Text == "" || in.Anchor.Quote == "" || !claimKinds[in.Kind] {
		return "", "", &domain.UserError{Msg: "A claim needs its text, a type and a quote from your notes."}
	}
	var page *int
	err = db.QueryRow(ctx, `SELECT b.source_id::text, b.page FROM source_blocks b
		JOIN pages p ON p.id = b.page_id JOIN uploads u ON u.id = p.upload_id
		WHERE b.id = $1 AND u.subject_id = $2 AND p.status = 'ready'`, in.Anchor.BlockID, subjectID).Scan(&sourceID, &page)
	if nf(err) == domain.ErrNotFound {
		return "", "", &domain.UserError{Msg: "That quote is not on a Ready page of this course."}
	}
	if page != nil {
		locator = "p." + strconv.Itoa(*page)
	}
	return
}

func (s *Store) CreateClaim(ctx context.Context, userID, conceptID string, in domain.ClaimInput) (domain.Claim, error) {
	sub, err := conceptSubject(ctx, s.pool, userID, conceptID)
	if err != nil {
		return domain.Claim{}, err
	}
	src, loc, err := anchorSource(ctx, s.pool, sub, in)
	if err != nil {
		return domain.Claim{}, err
	}
	var id string
	if err := s.pool.QueryRow(ctx, `INSERT INTO claims (concept_id, text, source_id, locator, required, kind, author_user_id, anchor_block_id, anchor_quote)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id::text`,
		conceptID, in.Text, src, loc, in.Required, in.Kind, userID, in.Anchor.BlockID, in.Anchor.Quote).Scan(&id); err != nil {
		return domain.Claim{}, err
	}
	return scanClaim(s.pool.QueryRow(ctx, `SELECT `+claimCols+` FROM claims c WHERE c.id = $1`, id))
}

func (s *Store) UpdateClaim(ctx context.Context, userID, id string, in domain.ClaimInput) (domain.Claim, error) {
	var conceptID string
	if err := s.pool.QueryRow(ctx, `SELECT c.concept_id::text FROM claims c WHERE c.id = $1 AND c.deleted_at IS NULL`, id).Scan(&conceptID); err != nil {
		return domain.Claim{}, nf(err)
	}
	sub, err := conceptSubject(ctx, s.pool, userID, conceptID)
	if err != nil {
		return domain.Claim{}, err
	}
	src, loc, err := anchorSource(ctx, s.pool, sub, in)
	if err != nil {
		return domain.Claim{}, err
	}
	if err := affected(s.pool.Exec(ctx, `UPDATE claims SET text = $2, source_id = $3, locator = $4, required = $5, kind = $6,
			anchor_block_id = $7, anchor_quote = $8 WHERE id = $1 AND deleted_at IS NULL`,
		id, in.Text, src, loc, in.Required, in.Kind, in.Anchor.BlockID, in.Anchor.Quote)); err != nil {
		return domain.Claim{}, err
	}
	return scanClaim(s.pool.QueryRow(ctx, `SELECT `+claimCols+` FROM claims c WHERE c.id = $1`, id))
}

func (s *Store) DeleteClaim(ctx context.Context, userID, id string) error {
	return affected(s.pool.Exec(ctx, `UPDATE claims c SET deleted_at = now() FROM concepts k JOIN subjects s ON s.id = k.subject_id
		WHERE c.id = $1 AND c.deleted_at IS NULL AND k.id = c.concept_id AND s.user_id = $2 AND s.archived_at IS NULL`, id, userID))
}
