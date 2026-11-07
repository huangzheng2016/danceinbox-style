package audit

import (
	"context"
	"encoding/json"
	"testing"

	"MODULE_NAME/biz/actor"
	"MODULE_NAME/biz/dal/db/dbtest"
	"MODULE_NAME/biz/dal/db/model"
)

func TestRecordTakesActorFromContextAndFilters(t *testing.T) {
	s := NewService(dbtest.New(t))
	alice := actor.With(context.Background(), 7)
	s.Record(alice, Entry{Action: model.AuditExampleCreate, TargetType: model.TargetExample, TargetID: 1, TargetName: "a",
		After: map[string]any{"name": "a"}})
	s.Record(alice, Entry{Action: model.AuditExampleUpdate, TargetType: model.TargetExample, TargetID: 1, TargetName: "a",
		Before: map[string]any{"name": "a"}, After: map[string]any{"name": "b"}})
	s.Record(context.Background(), Entry{Action: model.AuditExampleBatch, TargetType: model.TargetExample, Summary: "批量停用 2 项"})

	ctx := context.Background()
	all, total, err := s.List(ctx, ListParams{})
	if err != nil || total != 3 {
		t.Fatalf("got %d rows, err %v", total, err)
	}
	if all[0].Action != model.AuditExampleBatch || all[0].ActorID != 0 || all[0].Detail != "" {
		t.Fatalf("newest first, no actor, no detail: %+v", all[0])
	}

	rows, _, _ := s.List(ctx, ListParams{ActorID: 7, TargetType: model.TargetExample, TargetID: 1, Action: model.AuditExampleUpdate})
	if len(rows) != 1 {
		t.Fatalf("filters: got %d rows", len(rows))
	}
	var d struct{ Before, After map[string]any }
	if err := json.Unmarshal([]byte(rows[0].Detail), &d); err != nil || d.Before["name"] != "a" || d.After["name"] != "b" {
		t.Fatalf("detail: %s (%v)", rows[0].Detail, err)
	}
}
