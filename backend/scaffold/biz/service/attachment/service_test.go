package attachment

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"MODULE_NAME/biz/dal/db/dbtest"
	"MODULE_NAME/biz/dal/db/model"
	"MODULE_NAME/biz/errs"
	"MODULE_NAME/config"
)

func setup(t *testing.T) (*Service, int64) {
	t.Helper()
	config.C.Upload.Dir = t.TempDir()
	config.C.Upload.MaxMB = 1
	q := dbtest.New(t)
	ex := &model.Example{Name: "a", Status: model.StatusEnabled}
	_ = q.Example.WithContext(context.Background()).Create(ex)
	s, err := NewService(q)
	if err != nil {
		t.Fatal(err)
	}
	return s, ex.ID
}

func code(err error) int32 {
	var e *errs.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return 0
}

func TestSaveRejectsBadInput(t *testing.T) {
	s, exID := setup(t)
	ctx := context.Background()
	cases := []struct {
		name, file string
		body       io.Reader
	}{
		{"disallowed extension", "run.sh", strings.NewReader("echo")},
		{"no name", "", strings.NewReader("x")},
		{"empty file", "a.txt", strings.NewReader("")},
		{"over the limit", "big.txt", strings.NewReader(strings.Repeat("x", 1<<20+1))},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := s.Save(ctx, exID, 1, tc.file, tc.body); code(err) != errs.CodeInvalid {
				t.Fatalf("got %v", err)
			}
		})
	}
	entries, _ := os.ReadDir(config.C.Upload.Dir)
	if len(entries) != 0 {
		t.Fatalf("rejected uploads left %d files behind", len(entries))
	}
}

func TestSaveStoresUnderRandomNameAndRoundTrips(t *testing.T) {
	s, exID := setup(t)
	ctx := context.Background()
	row, err := s.Save(ctx, exID, 1, `..\..\etc/报告.CSV`, strings.NewReader("a,b\n1,2\n"))
	if err != nil {
		t.Fatal(err)
	}
	if row.Name != "报告.CSV" || row.Size != 8 || len(row.SHA256) != 64 {
		t.Fatalf("got %+v", row)
	}
	if strings.Contains(row.StoredName, ".") || filepath.Base(row.StoredName) != row.StoredName {
		t.Fatalf("stored name %q must not derive from the upload", row.StoredName)
	}
	_, f, err := s.Open(ctx, row.ID)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(f)
	f.Close()
	if string(body) != "a,b\n1,2\n" {
		t.Fatalf("content %q", body)
	}

	if err := s.Delete(ctx, row.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, row.ID); err != nil {
		t.Fatalf("deleting twice must succeed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(config.C.Upload.Dir, row.StoredName)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("file still on disk")
	}
}

func TestSaveRequiresExample(t *testing.T) {
	s, _ := setup(t)
	if _, err := s.Save(context.Background(), 999, 1, "a.txt", strings.NewReader("x")); code(err) != errs.CodeNotFound {
		t.Fatalf("got %v", err)
	}
}
