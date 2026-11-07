// Package settings is the reference for settings people edit at runtime: one
// typed document, saved as numbered versions. A save names the version it was
// edited from; if someone saved in between, it fails with a conflict instead of
// silently overwriting their change. Every version is kept, so any of them can
// be compared or restored.
//
// Settings vs config: a value belongs here when an admin should change it while
// the app runs and every service reads it per request (Current). Values the
// process needs at startup or that bound the server itself (listen address,
// secrets, upload.max_mb, which also caps the request body) stay in config.
package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"MODULE_NAME/biz/actor"
	"MODULE_NAME/biz/dal/db"
	"MODULE_NAME/biz/dal/db/model"
	"MODULE_NAME/biz/dal/db/query"
	"MODULE_NAME/biz/errs"
	"MODULE_NAME/biz/service/audit"

	"gorm.io/gorm"
)

// Settings is the whole document; each field is read by the service named next
// to it. Durations are stored in seconds; the page edits them in human units.
type Settings struct {
	// Markdown shown on the overview page (PublicSettings); "" shows nothing
	Announcement string `json:"announcement"`
	// run.Service: a run still going after this long is stopped as failed
	RunTimeoutSeconds int64 `json:"run_timeout_seconds"`
	// run.Service: while true, new runs are refused; running ones finish
	RunsPaused bool `json:"runs_paused"`
	// attachment.Service: per example; 0 means no limit
	MaxAttachments int64 `json:"max_attachments"`
}

// Defaults apply until the first save (version 0), and to fields a stored
// version predates.
func Defaults() Settings {
	return Settings{RunTimeoutSeconds: 300, MaxAttachments: 50}
}

type Snapshot struct {
	// 0 means never saved: Settings are the defaults
	Version      int64
	Settings     Settings
	RestoredFrom int64
	CreatedBy    int64
	CreatedAt    int64
}

type Service struct {
	q     *query.Query
	audit *audit.Service
}

func NewService(q *query.Query, a *audit.Service) *Service { return &Service{q: q, audit: a} }

func (s *Service) Current(ctx context.Context) (Snapshot, error) {
	v := s.q.SettingsVersion
	row, err := v.WithContext(ctx).Order(v.Version.Desc()).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Snapshot{Settings: Defaults()}, nil
	}
	if err != nil {
		return Snapshot{}, err
	}
	return decode(row)
}

func (s *Service) Version(ctx context.Context, version int64) (Snapshot, error) {
	if version == 0 {
		return Snapshot{Settings: Defaults()}, nil
	}
	v := s.q.SettingsVersion
	row, err := v.WithContext(ctx).Where(v.Version.Eq(version)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Snapshot{}, errs.NotFound("版本不存在")
	}
	if err != nil {
		return Snapshot{}, err
	}
	return decode(row)
}

// Versions lists saved versions, newest first.
func (s *Service) Versions(ctx context.Context, page, size int32) ([]Snapshot, int64, error) {
	v := s.q.SettingsVersion
	offset, limit := db.Page(page, size)
	rows, total, err := v.WithContext(ctx).Order(v.Version.Desc()).FindByPage(offset, limit)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Snapshot, len(rows))
	for i, r := range rows {
		if out[i], err = decode(r); err != nil {
			return nil, 0, err
		}
	}
	return out, total, nil
}

var ErrConflict = errs.Conflict("设置已被其他人修改，请查看差异后再保存")

// Save stores in as the version after base. restoredFrom names the version being
// restored (0 for an ordinary save) so history and the audit log can say so.
// Saving values equal to the current ones returns the current version unchanged.
func (s *Service) Save(ctx context.Context, base int64, in Settings, restoredFrom int64) (Snapshot, error) {
	in = normalize(in)
	if err := validate(in); err != nil {
		return Snapshot{}, err
	}
	cur, err := s.Current(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	if cur.Version != base {
		return Snapshot{}, ErrConflict
	}
	if reflect.DeepEqual(cur.Settings, in) {
		return cur, nil
	}
	data, err := json.Marshal(in)
	if err != nil {
		return Snapshot{}, err
	}
	row := &model.SettingsVersion{Version: base + 1, Data: string(data), RestoredFrom: restoredFrom, CreatedBy: actor.ID(ctx)}
	// two saves from the same base: the unique index lets exactly one through
	if err := s.q.SettingsVersion.WithContext(ctx).Create(row); errors.Is(err, gorm.ErrDuplicatedKey) {
		return Snapshot{}, ErrConflict
	} else if err != nil {
		return Snapshot{}, err
	}
	summary := fmt.Sprintf("保存为版本 %d", row.Version)
	if restoredFrom > 0 {
		summary = fmt.Sprintf("恢复版本 %d，保存为版本 %d", restoredFrom, row.Version)
	}
	s.audit.Record(ctx, audit.Entry{
		Action: model.AuditSettingsUpdate, TargetType: model.TargetSettings, TargetID: row.Version,
		TargetName: fmt.Sprintf("版本 %d", row.Version), Summary: summary, Before: cur.Settings, After: in,
	})
	return decode(row)
}

func normalize(in Settings) Settings {
	in.Announcement = strings.TrimSpace(in.Announcement)
	return in
}

func validate(in Settings) error {
	switch {
	case len([]rune(in.Announcement)) > 500:
		return errs.Invalid("公告不超过 500 个字符")
	case in.RunTimeoutSeconds < 10 || in.RunTimeoutSeconds > 86400:
		return errs.Invalid("运行超时需在 10 秒到 24 小时之间")
	case in.MaxAttachments < 0 || in.MaxAttachments > 1000:
		return errs.Invalid("附件数量上限需在 0 到 1000 之间")
	}
	return nil
}

func decode(row *model.SettingsVersion) (Snapshot, error) {
	st := Defaults()
	// decoding over the defaults gives fields added after this version their default
	if err := json.Unmarshal([]byte(row.Data), &st); err != nil {
		return Snapshot{}, fmt.Errorf("settings version %d: %w", row.Version, err)
	}
	return Snapshot{Version: row.Version, Settings: st, RestoredFrom: row.RestoredFrom, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt}, nil
}
