package settings

import (
	"context"
	"strings"
	"testing"

	"MODULE_NAME/biz/actor"
	"MODULE_NAME/biz/dal/db/dbtest"
	"MODULE_NAME/biz/dal/db/model"
	"MODULE_NAME/biz/errs"
	"MODULE_NAME/biz/service/audit"
)

func newService(t *testing.T) *Service {
	q := dbtest.New(t)
	return NewService(q, audit.NewService(q))
}

func codeOf(err error) int32 {
	if e, ok := err.(*errs.Error); ok {
		return e.Code
	}
	return 0
}

func TestDefaultsUntilFirstSave(t *testing.T) {
	s := newService(t)
	cur, err := s.Current(context.Background())
	if err != nil || cur.Version != 0 || cur.Settings != Defaults() {
		t.Fatalf("got %+v, %v", cur, err)
	}
}

func TestSaveVersionsAndRejectsStaleBase(t *testing.T) {
	ctx := actor.With(context.Background(), 3)
	s := newService(t)
	in := Defaults()
	in.Announcement = "  维护通知 "
	in.RunsPaused = true
	v1, err := s.Save(ctx, 0, in, 0)
	if err != nil {
		t.Fatal(err)
	}
	if v1.Version != 1 || v1.Settings.Announcement != "维护通知" || !v1.Settings.RunsPaused || v1.CreatedBy != 3 {
		t.Fatalf("normalized v1: %+v", v1)
	}

	// someone else edited from version 0 meanwhile
	if _, err := s.Save(ctx, 0, Defaults(), 0); err != ErrConflict {
		t.Fatalf("stale base: got %v", err)
	}
	// unchanged values do not create a version
	same, err := s.Save(ctx, 1, v1.Settings, 0)
	if err != nil || same.Version != 1 {
		t.Fatalf("unchanged save: %+v %v", same, err)
	}
	old, _ := s.Version(ctx, 0)
	v2, err := s.Save(ctx, 1, old.Settings, 0)
	if err != nil || v2.Version != 2 {
		t.Fatalf("v2: %+v %v", v2, err)
	}
	list, total, _ := s.Versions(ctx, 1, 20)
	if total != 2 || list[0].Version != 2 {
		t.Fatalf("versions: %d %+v", total, list)
	}
	logs, _, _ := s.audit.List(ctx, audit.ListParams{TargetType: model.TargetSettings})
	if len(logs) != 2 || logs[0].Summary != "保存为版本 2" {
		t.Fatalf("audit: %+v", logs)
	}
}

func TestValidate(t *testing.T) {
	s := newService(t)
	cases := map[string]func(*Settings){
		"long announcement": func(x *Settings) { x.Announcement = strings.Repeat("字", 501) },
		"short timeout":     func(x *Settings) { x.RunTimeoutSeconds = 5 },
		"negative limit":    func(x *Settings) { x.MaxAttachments = -1 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			in := Defaults()
			mutate(&in)
			if _, err := s.Save(context.Background(), 0, in, 0); codeOf(err) != errs.CodeInvalid {
				t.Fatalf("got %v", err)
			}
		})
	}
}
