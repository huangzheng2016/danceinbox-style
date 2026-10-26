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
