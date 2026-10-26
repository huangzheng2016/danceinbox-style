package example

import (
	"context"
	"fmt"
	"testing"
	"time"

	"MODULE_NAME/biz/dal/db/dbtest"
	"MODULE_NAME/biz/dal/db/model"
	"MODULE_NAME/biz/errs"
)

func TestCreateValidatesAndRejectsDuplicates(t *testing.T) {
	ctx := context.Background()
	s := NewService(dbtest.New(t))
	if _, err := s.Create(ctx, Input{Name: "a", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		in   Input
		code int32
	}{
		{"blank name", Input{Name: "  "}, errs.CodeInvalid},
		{"duplicate name", Input{Name: "a"}, errs.CodeConflict},
		{"duplicate after trim", Input{Name: " a "}, errs.CodeConflict},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := s.Create(ctx, tc.in); codeOf(err) != tc.code {
				t.Fatalf("got %v, want code %d", err, tc.code)
			}
		})
	}
}

func TestUpdateWritesZeroValuesAndUnchangedRows(t *testing.T) {
	ctx := context.Background()
	s := NewService(dbtest.New(t))
	row, _ := s.Create(ctx, Input{Name: "a", Remark: "note", Enabled: true})

	got, err := s.Update(ctx, row.ID, Input{Name: "a", Remark: "", Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	if got.Remark != "" || got.Status != "disabled" {
		t.Fatalf("zero values not written: %+v", got)
	}
	if _, err := s.Update(ctx, row.ID, Input{Name: "a"}); err != nil {
		t.Fatalf("saving unchanged values must succeed: %v", err)
	}
	if _, err := s.Update(ctx, 999, Input{Name: "x"}); codeOf(err) != errs.CodeNotFound {
		t.Fatalf("got %v, want not found", err)
	}
}

func TestDeleteRequiresDisabled(t *testing.T) {
	ctx := context.Background()
	s := NewService(dbtest.New(t))
	row, _ := s.Create(ctx, Input{Name: "a", Enabled: true})

	if err := s.Delete(ctx, row.ID); codeOf(err) != errs.CodeConflict {
		t.Fatalf("deleting an enabled row: got %v, want conflict", err)
	}
	if err := s.SetEnabled(ctx, row.ID, false); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, row.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, row.ID); err != nil {
		t.Fatalf("deleting twice must succeed: %v", err)
	}
}

func TestListFiltersAndPages(t *testing.T) {
	ctx := context.Background()
	s := NewService(dbtest.New(t))
	for _, n := range []string{"alpha", "beta", "al_pha", "gamma"} {
		_, _ = s.Create(ctx, Input{Name: n, Enabled: n != "gamma"})
	}
	cases := []struct {
		name  string
		p     ListParams
		total int64
		rows  int
	}{
		{"all, default size", ListParams{}, 4, 4},
		{"substring", ListParams{Q: "al"}, 2, 2},
		{"underscore is literal", ListParams{Q: "l_p"}, 1, 1},
		{"status", ListParams{Status: "disabled"}, 1, 1},
		{"second page", ListParams{Page: 2, Size: 3}, 4, 1},
		{"size above max is clamped", ListParams{Size: 1000}, 4, 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows, total, err := s.List(ctx, tc.p)
			if err != nil || total != tc.total || len(rows) != tc.rows {
				t.Fatalf("got %d rows, total %d, err %v; want %d, %d", len(rows), total, err, tc.rows, tc.total)
			}
		})
	}
}

func TestListSortsAndFiltersByTime(t *testing.T) {
	ctx := context.Background()
	q := dbtest.New(t)
	s := NewService(q)
	for i, n := range []string{"b", "c", "a"} {
		row, _ := s.Create(ctx, Input{Name: n})
		// spread updated_at so the range filter has something to cut
		_, _ = q.Example.WithContext(ctx).Where(q.Example.ID.Eq(row.ID)).UpdateColumnSimple(q.Example.UpdatedAt.Value(int64(1000 * (i + 1))))
	}
	names := func(p ListParams) string {
		rows, _, err := s.List(ctx, p)
		if err != nil {
			return err.Error()
		}
		out := ""
		for _, r := range rows {
			out += r.Name
		}
		return out
	}
	cases := []struct {
		name string
		p    ListParams
		want string
	}{
		{"default newest first", ListParams{}, "acb"},
		{"by name", ListParams{Sort: "name"}, "abc"},
		{"by name desc", ListParams{Sort: "-name"}, "cba"},
		{"time range is half-open", ListParams{UpdatedFrom: 1000, UpdatedTo: 3000}, "cb"},
		{"unknown sort field", ListParams{Sort: "remark"}, "不支持按该字段排序"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := names(tc.p); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBatchReportsDoneAndSkipped(t *testing.T) {
	ctx := context.Background()
	s := NewService(dbtest.New(t))
	a, _ := s.Create(ctx, Input{Name: "a", Enabled: true})
	b, _ := s.Create(ctx, Input{Name: "b"})
	ids := []int64{a.ID, b.ID, 999}

	if done, skipped, err := s.Batch(ctx, ids, "enable"); err != nil || done != 2 || skipped != 1 {
		t.Fatalf("enable: done %d skipped %d err %v", done, skipped, err)
	}
	_ = s.SetEnabled(ctx, b.ID, false)
	if done, skipped, err := s.Batch(ctx, ids, "delete"); err != nil || done != 1 || skipped != 2 {
		t.Fatalf("delete skips enabled and missing rows: done %d skipped %d err %v", done, skipped, err)
	}
	if _, err := s.Get(ctx, a.ID); err != nil {
		t.Fatalf("enabled row must survive a batch delete: %v", err)
	}
}

func TestOwnerMustBeActiveUser(t *testing.T) {
	ctx := context.Background()
	q := dbtest.New(t)
	s := NewService(q)
	owner := &model.User{Username: "o", PasswordHash: "x", Active: true}
	_ = q.User.WithContext(ctx).Create(owner)

	if _, err := s.Create(ctx, Input{Name: "a", OwnerID: owner.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, Input{Name: "b", OwnerID: 999}); codeOf(err) != errs.CodeInvalid {
		t.Fatalf("got %v, want invalid", err)
	}
}

func TestImportReportsFailedRows(t *testing.T) {
	ctx := context.Background()
	s := NewService(dbtest.New(t))
	_, _ = s.Create(ctx, Input{Name: "exists"})
	created, failed, err := s.Import(ctx, []Input{
		{Name: "a"},
		{Name: "exists"},
		{Name: "b", Config: "[1,2]"},
		{Name: "a"},
		{Name: "c", Config: `{"k":1}`},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created != 2 {
		t.Fatalf("created %d", created)
	}
	want := []ImportFailure{{2, "名称已存在"}, {3, "配置必须是 JSON 对象"}, {4, "名称已存在"}}
	if len(failed) != len(want) {
		t.Fatalf("failed %+v", failed)
	}
	for i := range want {
		if failed[i] != want[i] {
			t.Fatalf("failure %d: got %+v, want %+v", i, failed[i], want[i])
		}
	}
}

func TestStatsFillsEveryDay(t *testing.T) {
	ctx := context.Background()
	q := dbtest.New(t)
	s := NewService(q)
	now := time.Date(2026, 10, 26, 15, 0, 0, 0, time.Local)
	for _, at := range []time.Time{now, now.Add(-2 * time.Hour), now.AddDate(0, 0, -2), now.AddDate(0, 0, -30)} {
		row, _ := s.Create(ctx, Input{Name: at.String()})
		_, _ = q.Example.WithContext(ctx).Where(q.Example.ID.Eq(row.ID)).UpdateColumnSimple(q.Example.CreatedAt.Value(at.Unix()))
	}
	days, err := s.Stats(ctx, 3, now)
	if err != nil {
		t.Fatal(err)
	}
	got := fmt.Sprint(days)
	if want := "[{2026-10-24 1} {2026-10-25 0} {2026-10-26 2}]"; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
