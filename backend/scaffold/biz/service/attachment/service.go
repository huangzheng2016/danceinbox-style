// Package attachment stores uploaded files on local disk under config upload.dir.
// Stored names are random; the user's file name is only ever display text.
package attachment

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"MODULE_NAME/biz/dal/db/model"
	"MODULE_NAME/biz/dal/db/query"
	"MODULE_NAME/biz/errs"
	"MODULE_NAME/config"

	"gorm.io/gorm"
)

// Extensions accepted for upload. Downloads are always served as attachments,
// so even an allowed HTML-like file is never rendered by the browser.
var allowedExt = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
	".pdf": true, ".txt": true, ".csv": true, ".json": true, ".md": true,
	".zip": true, ".xlsx": true, ".docx": true,
}

const maxPerExample = 50

type Service struct {
	q   *query.Query
	dir string
	max int64
}

func NewService(q *query.Query) (*Service, error) {
	dir := config.C.Upload.Dir
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	return &Service{q: q, dir: dir, max: int64(config.C.Upload.MaxMB) << 20}, nil
}

// MaxBytes is the per-file limit, for the handler's early check and error message.
func (s *Service) MaxBytes() int64 { return s.max }

// Save streams r to disk while hashing it, then records it. A partially written
// file is removed on any failure.
func (s *Service) Save(ctx context.Context, exampleID, userID int64, name string, r io.Reader) (*model.Attachment, error) {
	name, err := cleanName(name)
	if err != nil {
		return nil, err
	}
	e := s.q.Example
	exists, err := e.WithContext(ctx).Where(e.ID.Eq(exampleID)).Count()
	if err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, errs.NotFound("示例不存在")
	}
	a := s.q.Attachment
	count, err := a.WithContext(ctx).Where(a.ExampleID.Eq(exampleID)).Count()
	if err != nil {
		return nil, err
	}
	if count >= maxPerExample {
		return nil, errs.Conflict(fmt.Sprintf("每个示例最多 %d 个附件", maxPerExample))
	}

	stored, err := randomName()
	if err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp(s.dir, ".upload-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	hash := sha256.New()
	head := make([]byte, 512)
	n, _ := io.ReadFull(r, head)
	head = head[:n]
	size, err := io.Copy(io.MultiWriter(tmp, hash), io.LimitReader(io.MultiReader(bytes.NewReader(head), r), s.max+1))
	if err != nil {
		return nil, err
	}
	if size > s.max {
		return nil, errs.Invalid(fmt.Sprintf("文件超过 %d MB", s.max>>20))
	}
	if size == 0 {
		return nil, errs.Invalid("文件为空")
	}
	if err := tmp.Close(); err != nil {
		return nil, err
	}
	if err := os.Rename(tmp.Name(), filepath.Join(s.dir, stored)); err != nil {
		return nil, err
	}

	row := &model.Attachment{
		ExampleID: exampleID, Name: name, StoredName: stored, Size: size,
		ContentType: http.DetectContentType(head), SHA256: hex.EncodeToString(hash.Sum(nil)), CreatedBy: userID,
	}
	if err := a.WithContext(ctx).Create(row); err != nil {
		os.Remove(filepath.Join(s.dir, stored))
		return nil, err
	}
	return row, nil
}

func (s *Service) List(ctx context.Context, exampleID int64) ([]*model.Attachment, error) {
	a := s.q.Attachment
	return a.WithContext(ctx).Where(a.ExampleID.Eq(exampleID)).Order(a.ID.Desc()).Find()
}

// Open returns the record and its file for download. The caller closes the file.
func (s *Service) Open(ctx context.Context, id int64) (*model.Attachment, *os.File, error) {
	row, err := s.get(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	f, err := os.Open(filepath.Join(s.dir, row.StoredName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil, errs.NotFound("文件已丢失")
	}
	return row, f, err
}

// Delete removes the record first, then the file: a crash in between leaves an
// orphan file, never a record pointing at nothing. Deleting twice succeeds.
func (s *Service) Delete(ctx context.Context, id int64) error {
	row, err := s.get(ctx, id)
	var e *errs.Error
	if errors.As(err, &e) && e.Code == errs.CodeNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	a := s.q.Attachment
	if _, err := a.WithContext(ctx).Where(a.ID.Eq(id)).Delete(); err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(s.dir, row.StoredName)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *Service) get(ctx context.Context, id int64) (*model.Attachment, error) {
	a := s.q.Attachment
	row, err := a.WithContext(ctx).Where(a.ID.Eq(id)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.NotFound("附件不存在")
	}
	return row, err
}

// cleanName keeps the base name only and rejects names the UI could not show.
func cleanName(name string) (string, error) {
	name = strings.TrimSpace(filepath.Base(strings.ReplaceAll(name, `\`, "/")))
	if name == "" || name == "." || name == "/" || !utf8.ValidString(name) {
		return "", errs.Invalid("文件名无效")
	}
	if len([]rune(name)) > 200 {
		return "", errs.Invalid("文件名过长")
	}
	if !allowedExt[strings.ToLower(filepath.Ext(name))] {
		return "", errs.Invalid("不支持的文件类型:" + filepath.Ext(name))
	}
	return name, nil
}

func randomName() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
