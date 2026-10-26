// Package example is the reference feature. New features copy its shape:
// a struct over *query.Query, typed conditions only, errs for anything the
// client should see.
package example

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"MODULE_NAME/biz/dal/db"
	"MODULE_NAME/biz/dal/db/model"
	"MODULE_NAME/biz/dal/db/query"
	"MODULE_NAME/biz/errs"

	"gorm.io/gen/field"
	"gorm.io/gorm"
)

type Service struct{ q *query.Query }

func NewService(q *query.Query) *Service { return &Service{q: q} }

type ListParams struct {
	Q           string
	Status      string
	Sort        string
	UpdatedFrom int64
	UpdatedTo   int64
	Page        int32
	Size        int32
}

func (s *Service) List(ctx context.Context, p ListParams) ([]*model.Example, int64, error) {
	e := s.q.Example
	// Only these columns are sortable: each needs an index to stay fast.
	order, err := db.Order(p.Sort, map[string]field.OrderExpr{
		"name":       e.Name,
		"created_at": e.CreatedAt,
		"updated_at": e.UpdatedAt,
	})
	if err != nil {
		return nil, 0, err
	}
	if order == nil {
		order = e.UpdatedAt.Desc()
	}
	do := e.WithContext(ctx)
	if q := strings.TrimSpace(p.Q); q != "" {
		do = do.Where(db.Contains(e.Name, q))
	}
	if p.Status != "" {
		do = do.Where(e.Status.Eq(p.Status))
	}
	if p.UpdatedFrom > 0 {
		do = do.Where(e.UpdatedAt.Gte(p.UpdatedFrom))
	}
	if p.UpdatedTo > 0 {
		do = do.Where(e.UpdatedAt.Lt(p.UpdatedTo))
	}
	offset, limit := db.Page(p.Page, p.Size)
	return do.Order(order, e.ID.Desc()).FindByPage(offset, limit)
}

func (s *Service) Get(ctx context.Context, id int64) (*model.Example, error) {
	e := s.q.Example
	row, err := e.WithContext(ctx).Where(e.ID.Eq(id)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.NotFound("示例不存在")
	}
	return row, err
}

type Input struct {
	Name    string
	Remark  string
	Enabled bool
	// 0 means no owner
	OwnerID int64
	// a JSON object, or "" for none
	Config string
}

func (s *Service) validate(ctx context.Context, in Input) error {
	if strings.TrimSpace(in.Name) == "" {
		return errs.Invalid("名称不能为空")
	}
	if len([]rune(in.Name)) > 64 {
		return errs.Invalid("名称不超过 64 个字符")
	}
	if len(in.Remark) > 20000 {
		return errs.Invalid("备注过长")
	}
	if in.Config != "" {
		var obj map[string]any
		if err := json.Unmarshal([]byte(in.Config), &obj); err != nil {
			return errs.Invalid("配置必须是 JSON 对象")
		}
	}
	if in.OwnerID != 0 {
		u := s.q.User
		n, err := u.WithContext(ctx).Where(u.ID.Eq(in.OwnerID), u.Active.Is(true)).Count()
		if err != nil {
			return err
		}
		if n == 0 {
			return errs.Invalid("负责人不存在或已停用")
		}
	}
	return nil
}

func (s *Service) Create(ctx context.Context, in Input) (*model.Example, error) {
	if err := s.validate(ctx, in); err != nil {
		return nil, err
	}
	row := &model.Example{
		Name: strings.TrimSpace(in.Name), Remark: in.Remark, Status: status(in.Enabled),
		OwnerID: in.OwnerID, Config: in.Config,
	}
	if err := s.q.Example.WithContext(ctx).Create(row); err != nil {
		return nil, conflict(err)
	}
	return row, nil
}

// RowsAffected is not used for existence: MySQL reports 0 when the new values
// equal the old ones. Check first, then write.
func (s *Service) Update(ctx context.Context, id int64, in Input) (*model.Example, error) {
	if err := s.validate(ctx, in); err != nil {
		return nil, err
	}
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	e := s.q.Example
	// UpdateSimple writes zero values too (an emptied remark), unlike Updates(struct).
	_, err := e.WithContext(ctx).Where(e.ID.Eq(id)).UpdateSimple(
		e.Name.Value(strings.TrimSpace(in.Name)),
		e.Remark.Value(in.Remark),
		e.Status.Value(status(in.Enabled)),
		e.OwnerID.Value(in.OwnerID),
		e.Config.Value(in.Config),
	)
	if err != nil {
		return nil, conflict(err)
	}
	return s.Get(ctx, id)
}

func (s *Service) SetEnabled(ctx context.Context, id int64, enabled bool) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	e := s.q.Example
	_, err := e.WithContext(ctx).Where(e.ID.Eq(id)).UpdateSimple(e.Status.Value(status(enabled)))
	return err
}

// Delete only removes disabled examples, so an enabled one cannot vanish by
// accident. The state check is part of the WHERE clause, not a separate read.
// Deleting a row that is already gone succeeds, so retries are safe.
func (s *Service) Delete(ctx context.Context, id int64) error {
	e := s.q.Example
	res, err := e.WithContext(ctx).Where(e.ID.Eq(id), e.Status.Eq(model.StatusDisabled)).Delete()
	if err != nil || res.RowsAffected > 0 {
		return err
	}
	_, err = s.Get(ctx, id)
	if codeOf(err) == errs.CodeNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	return errs.Conflict("请先停用再删除")
}

// Batch applies action to every id. Ids that do not exist, and enabled rows for
// "delete", are skipped rather than failing the whole batch; the caller reports both counts.
func (s *Service) Batch(ctx context.Context, ids []int64, action string) (done, skipped int, err error) {
	e := s.q.Example
	scope := e.WithContext(ctx).Where(e.ID.In(ids...))
	switch action {
	case "enable", "disable":
		// Count first: MySQL's RowsAffected skips rows that already had the value.
		n, err := scope.Count()
		if err != nil {
			return 0, 0, err
		}
		if _, err := scope.UpdateSimple(e.Status.Value(status(action == "enable"))); err != nil {
			return 0, 0, err
		}
		done = int(n)
	case "delete":
		res, err := scope.Where(e.Status.Eq(model.StatusDisabled)).Delete()
		if err != nil {
			return 0, 0, err
		}
		done = int(res.RowsAffected)
	default:
		return 0, 0, errs.Invalid("不支持的批量操作")
	}
	return done, len(ids) - done, nil
}

type ImportFailure struct {
	Row    int
	Reason string
}

// Import creates rows one by one and reports each failure with its 1-based row
// number, so the user can fix exactly those rows and import them again.
func (s *Service) Import(ctx context.Context, rows []Input) (created int, failed []ImportFailure, err error) {
	for i, in := range rows {
		_, err := s.Create(ctx, in)
		var e *errs.Error
		switch {
		case err == nil:
			created++
		case errors.As(err, &e):
			failed = append(failed, ImportFailure{Row: i + 1, Reason: e.Msg})
		default:
			return created, failed, err
		}
	}
	return created, failed, nil
}

type DayCount struct {
	Date    string
	Created int
}

// Stats counts examples created per day over the last `days` days, oldest first,
// days without any included. Bucketing happens in Go so it works on every
// dialect; past a few hundred thousand rows, move it into a SQL aggregate in
// biz/dal/db.
func (s *Service) Stats(ctx context.Context, days int, now time.Time) ([]DayCount, error) {
	if days < 1 || days > 90 {
		return nil, errs.Invalid("天数需在 1 到 90 之间")
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	start := today.AddDate(0, 0, -(days - 1))
	e := s.q.Example
	rows, err := e.WithContext(ctx).Select(e.CreatedAt).Where(e.CreatedAt.Gte(start.Unix())).Find()
	if err != nil {
		return nil, err
	}
	out := make([]DayCount, days)
	index := map[string]int{}
	for i := range out {
		out[i].Date = start.AddDate(0, 0, i).Format("2006-01-02")
		index[out[i].Date] = i
	}
	for _, r := range rows {
		if i, ok := index[time.Unix(r.CreatedAt, 0).In(now.Location()).Format("2006-01-02")]; ok {
			out[i].Created++
		}
	}
	return out, nil
}

func status(enabled bool) string {
	if enabled {
		return model.StatusEnabled
	}
	return model.StatusDisabled
}

func codeOf(err error) int32 {
	var e *errs.Error
	if errors.As(err, &e) {
		return e.Code
	}
	return 0
}

func conflict(err error) error {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return errs.Conflict("名称已存在")
	}
	return err
}
