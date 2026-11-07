// Package audit records who changed what. Services call Record after a change
// succeeds; the actor comes from the request context (biz/actor).
package audit

import (
	"context"
	"encoding/json"

	"MODULE_NAME/biz/actor"
	"MODULE_NAME/biz/dal/db"
	"MODULE_NAME/biz/dal/db/model"
	"MODULE_NAME/biz/dal/db/query"

	"github.com/cloudwego/hertz/pkg/common/hlog"
)

type Service struct{ q *query.Query }

func NewService(q *query.Query) *Service { return &Service{q: q} }

type Entry struct {
	Action     string
	TargetType string
	TargetID   int64
	TargetName string
	Summary    string
	// user-visible fields before and after the change (a map or a struct with json
	// tags), keyed as the frontend's AUDIT_FIELDS expect; left nil when the change
	// has no "before" (create) or "after" (delete)
	Before any
	After  any
}

// Record writes one entry. A failure is logged, not returned: the change itself
// already happened, and telling the user it failed would be wrong.
func (s *Service) Record(ctx context.Context, e Entry) {
	detail := ""
	if e.Before != nil || e.After != nil {
		b, err := json.Marshal(map[string]any{"before": e.Before, "after": e.After})
		if err != nil {
			hlog.CtxErrorf(ctx, "audit %s: encode detail: %v", e.Action, err)
		}
		detail = string(b)
	}
	row := &model.AuditLog{
		ActorID: actor.ID(ctx), Action: e.Action, TargetType: e.TargetType, TargetID: e.TargetID,
		TargetName: e.TargetName, Summary: e.Summary, Detail: detail,
	}
	if err := s.q.AuditLog.WithContext(ctx).Create(row); err != nil {
		hlog.CtxErrorf(ctx, "audit %s %s#%d: %v", e.Action, e.TargetType, e.TargetID, err)
	}
}

type ListParams struct {
	ActorID    int64
	Action     string
	TargetType string
	// 0 means any target of TargetType
	TargetID int64
	// created_at within [From, To), unix seconds; 0 means unbounded
	From int64
	To   int64
	Page int32
	Size int32
}

// List is newest first. Every filter is optional.
func (s *Service) List(ctx context.Context, p ListParams) ([]*model.AuditLog, int64, error) {
	a := s.q.AuditLog
	do := a.WithContext(ctx)
	if p.ActorID != 0 {
		do = do.Where(a.ActorID.Eq(p.ActorID))
	}
	if p.Action != "" {
		do = do.Where(a.Action.Eq(p.Action))
	}
	if p.TargetType != "" {
		do = do.Where(a.TargetType.Eq(p.TargetType))
	}
	if p.TargetID != 0 {
		do = do.Where(a.TargetID.Eq(p.TargetID))
	}
	if p.From > 0 {
		do = do.Where(a.CreatedAt.Gte(p.From))
	}
	if p.To > 0 {
		do = do.Where(a.CreatedAt.Lt(p.To))
	}
	offset, limit := db.Page(p.Page, p.Size)
	return do.Order(a.CreatedAt.Desc(), a.ID.Desc()).FindByPage(offset, limit)
}
