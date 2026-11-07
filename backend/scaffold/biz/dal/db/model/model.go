// Package model is the hand-written schema. cmd/generate reads it to emit the
// typed query package, and db.Init migrates it at startup.
package model

// All lists every table the app owns. Add each new model here once.
func All() []any {
	return []any{
		&User{},
		&Example{},
		&Run{},
		&RunEvent{},
		&Attachment{},
		&AuditLog{},
		&SettingsVersion{},
	}
}

type User struct {
	// No explicit type on the key: GORM then picks an auto-increment column on both MySQL and SQLite.
	ID           int64  `gorm:"primaryKey;autoIncrement"`
	Username     string `gorm:"type:varchar(64);not null;uniqueIndex"`
	Email        string `gorm:"type:varchar(255);not null;default:''"`
	PasswordHash string `gorm:"type:varchar(100);not null"`
	Role         string `gorm:"type:varchar(16);not null;default:USER"`
	Active       bool   `gorm:"not null;default:true"`
	CreatedAt    int64  `gorm:"autoCreateTime"`
	UpdatedAt    int64  `gorm:"autoUpdateTime"`
}

type Example struct {
	ID     int64  `gorm:"primaryKey;autoIncrement"`
	Name   string `gorm:"type:varchar(64);not null;uniqueIndex"`
	Status string `gorm:"type:varchar(16);not null;index"`
	// Markdown
	Remark  string `gorm:"type:text;not null"`
	OwnerID int64  `gorm:"not null;default:0;index"`
	// JSON object, validated by the service; empty means none
	Config    string `gorm:"type:text;not null"`
	CreatedAt int64  `gorm:"autoCreateTime"`
	UpdatedAt int64  `gorm:"autoUpdateTime"`
}

// Run is one execution of an agent against an example. Its output is the
// ordered RunEvent stream, which clients replay and follow.
type Run struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	ExampleID int64  `gorm:"not null;index"`
	Status    string `gorm:"type:varchar(16);not null;index"`
	Error     string `gorm:"type:varchar(1000);not null;default:''"`
	CreatedBy int64  `gorm:"not null"`
	CreatedAt int64  `gorm:"autoCreateTime"`
	EndedAt   int64  `gorm:"not null;default:0"`
}

type RunEvent struct {
	ID    int64  `gorm:"primaryKey;autoIncrement"`
	RunID int64  `gorm:"not null;uniqueIndex:idx_run_seq"`
	Seq   int64  `gorm:"not null;uniqueIndex:idx_run_seq"`
	Type  string `gorm:"type:varchar(16);not null"`
	Data  string `gorm:"type:text;not null"`
}

type Attachment struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	ExampleID int64  `gorm:"not null;index"`
	Name      string `gorm:"type:varchar(255);not null"`
	// random file name under config upload.dir; never derived from user input
	StoredName  string `gorm:"type:varchar(64);not null;uniqueIndex"`
	Size        int64  `gorm:"not null"`
	ContentType string `gorm:"type:varchar(128);not null"`
	SHA256      string `gorm:"type:char(64);not null"`
	CreatedBy   int64  `gorm:"not null"`
	CreatedAt   int64  `gorm:"autoCreateTime"`
}

// AuditLog is one change made by a person. Written by services after the change
// succeeds (see biz/service/audit); never updated or deleted by the app.
type AuditLog struct {
	ID int64 `gorm:"primaryKey;autoIncrement"`
	// 0 for changes made without a session (startup jobs, CLI)
	ActorID    int64  `gorm:"not null;default:0;index"`
	Action     string `gorm:"type:varchar(32);not null;index"`
	TargetType string `gorm:"type:varchar(32);not null;index:idx_audit_target"`
	// 0 when the change has no single target (batch, import)
	TargetID int64 `gorm:"not null;default:0;index:idx_audit_target"`
	// the target's name at the time, so the log still reads after a rename or delete
	TargetName string `gorm:"type:varchar(255);not null;default:''"`
	// one Chinese sentence for changes a diff cannot show ("批量停用 3 项，跳过 2 项")
	Summary string `gorm:"type:varchar(500);not null;default:''"`
	// JSON {"before": {...}, "after": {...}}, or "" when there is nothing to compare
	Detail    string `gorm:"type:text;not null"`
	CreatedAt int64  `gorm:"autoCreateTime;index"`
}

// SettingsVersion is one saved state of the app settings. Rows are only ever
// appended; the highest Version is current. The unique index makes two
// concurrent saves from the same base version fail instead of both winning.
type SettingsVersion struct {
	ID      int64 `gorm:"primaryKey;autoIncrement"`
	Version int64 `gorm:"not null;uniqueIndex"`
	// JSON of settings.Settings
	Data string `gorm:"type:text;not null"`
	// the version this one restored, 0 for an ordinary save
	RestoredFrom int64 `gorm:"not null;default:0"`
	CreatedBy    int64 `gorm:"not null"`
	CreatedAt    int64 `gorm:"autoCreateTime"`
}

const (
	RunQueued    = "queued"
	RunRunning   = "running"
	RunSucceeded = "succeeded"
	RunFailed    = "failed"
	RunCanceled  = "canceled"
)

const (
	RoleUser      = "USER"
	RoleModerator = "MODERATOR"
	RoleAdmin     = "ADMIN"

	StatusEnabled  = "enabled"
	StatusDisabled = "disabled"
)

// Audit actions and target types. The frontend shows them through labels.ts
// (AUDIT_ACTION, AUDIT_TARGET); a new one is added in both places.
const (
	AuditExampleCreate  = "example.create"
	AuditExampleUpdate  = "example.update"
	AuditExampleEnable  = "example.enable"
	AuditExampleDisable = "example.disable"
	AuditExampleDelete  = "example.delete"
	AuditExampleBatch   = "example.batch"
	AuditExampleImport  = "example.import"
	AuditSettingsUpdate = "settings.update"

	TargetExample  = "example"
	TargetSettings = "settings"
)
