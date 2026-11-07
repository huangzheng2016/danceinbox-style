package papi

import (
	"context"

	"MODULE_NAME/biz/dal/db/model"
	audit "MODULE_NAME/biz/model/api/audit"
	settings "MODULE_NAME/biz/model/api/settings"
	svc "MODULE_NAME/biz/service/settings"
	"MODULE_NAME/container"
)

func toAppSettings(s svc.Settings) *settings.AppSettings {
	return &settings.AppSettings{
		Announcement: s.Announcement, RunTimeoutSeconds: s.RunTimeoutSeconds,
		RunsPaused: s.RunsPaused, MaxAttachments: s.MaxAttachments,
	}
}

func fromAppSettings(s *settings.AppSettings) svc.Settings {
	if s == nil {
		return svc.Settings{}
	}
	return svc.Settings{
		Announcement: s.Announcement, RunTimeoutSeconds: s.RunTimeoutSeconds,
		RunsPaused: s.RunsPaused, MaxAttachments: s.MaxAttachments,
	}
}

// toSnapshots resolves who saved each version with one query.
func toSnapshots(ctx context.Context, snaps ...svc.Snapshot) ([]*settings.SettingsSnapshot, error) {
	ids := make([]int64, 0, len(snaps))
	for _, s := range snaps {
		if s.CreatedBy != 0 {
			ids = append(ids, s.CreatedBy)
		}
	}
	names, err := container.Default.User.Names(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]*settings.SettingsSnapshot, len(snaps))
	for i, s := range snaps {
		out[i] = &settings.SettingsSnapshot{
			Version: s.Version, Settings: toAppSettings(s.Settings), RestoredFrom: s.RestoredFrom,
			UpdatedByName: names[s.CreatedBy], UpdatedAt: s.CreatedAt,
		}
	}
	return out, nil
}

// toAuditLogs is the same as the api package's; see there.
func toAuditLogs(ctx context.Context, rows []*model.AuditLog) ([]*audit.AuditLog, error) {
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		if r.ActorID != 0 {
			ids = append(ids, r.ActorID)
		}
	}
	names, err := container.Default.User.Names(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]*audit.AuditLog, len(rows))
	for i, r := range rows {
		out[i] = &audit.AuditLog{
			ID: r.ID, ActorID: r.ActorID, ActorName: names[r.ActorID], Action: r.Action,
			TargetType: r.TargetType, TargetID: r.TargetID, TargetName: r.TargetName,
			Summary: r.Summary, Detail: r.Detail, CreatedAt: r.CreatedAt,
		}
	}
	return out, nil
}
