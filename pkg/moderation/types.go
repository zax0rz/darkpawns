// Package moderation implements abuse reporting and admin controls for Dark Pawns.
package moderation

import (
	"time"
)

// ReportType represents the type of abuse being reported.
type ReportType string

const (
	ReportTypeHarassment ReportType = "harassment"
	ReportTypeSpam       ReportType = "spam"
	ReportTypeCheating   ReportType = "cheating"
	ReportTypeHateSpeech ReportType = "hate_speech"
	ReportTypeExploit    ReportType = "exploit"
	ReportTypeOther      ReportType = "other"
)

// ReportStatus represents the status of a report.
type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusReviewed  ReportStatus = "reviewed"
	ReportStatusResolved  ReportStatus = "resolved"
	ReportStatusDismissed ReportStatus = "dismissed"
)

// AbuseReport represents a player report of abusive behavior.
type AbuseReport struct {
	ID          int          `json:"id"`
	Reporter    string       `json:"reporter"` // Player who made the report
	Target      string       `json:"target"`   // Player being reported
	ReportType  ReportType   `json:"report_type"`
	Description string       `json:"description"`
	RoomVNum    int          `json:"room_vnum"` // Where it happened
	Timestamp   time.Time    `json:"timestamp"`
	Status      ReportStatus `json:"status"`
	ReviewedBy  string       `json:"reviewed_by"` // Admin who reviewed
	ReviewedAt  *time.Time   `json:"reviewed_at"`
	Resolution  string       `json:"resolution"` // What action was taken
}

// AdminAction represents an action taken by an admin.
type AdminAction string

const (
	ActionWarn        AdminAction = "warn"
	ActionMute        AdminAction = "mute"
	ActionKick        AdminAction = "kick"
	ActionBan         AdminAction = "ban"
	ActionInvestigate AdminAction = "investigate"
)

// AdminLogEntry logs an admin action for audit purposes.
type AdminLogEntry struct {
	ID        int            `json:"id"`
	Admin     string         `json:"admin"` // Admin who performed action
	Action    AdminAction    `json:"action"`
	Target    string         `json:"target"` // Player affected
	Reason    string         `json:"reason"`
	Duration  *time.Duration `json:"duration"` // For temporary actions (mute, ban)
	Timestamp time.Time      `json:"timestamp"`
	IPAddress string         `json:"ip_address"` // Optional: IP of target
}

// PlayerPenalty tracks active penalties on players.
type PlayerPenalty struct {
	PlayerName  string      `json:"player_name"`
	PenaltyType AdminAction `json:"penalty_type"`
	IssuedAt    time.Time   `json:"issued_at"`
	ExpiresAt   *time.Time  `json:"expires_at"` // nil for permanent
	Reason      string      `json:"reason"`
	IssuedBy    string      `json:"issued_by"`
}

// WordFilterEntry represents a filtered word or phrase.
type WordFilterEntry struct {
	ID        int          `json:"id"`
	Pattern   string       `json:"pattern"` // Regex pattern or exact match
	IsRegex   bool         `json:"is_regex"`
	Action    FilterAction `json:"action"` // What to do when matched
	CreatedBy string       `json:"created_by"`
	CreatedAt time.Time    `json:"created_at"`
}

// FilterAction represents what to do when a filtered word is detected.
type FilterAction string

const (
	FilterActionCensor FilterAction = "censor" // Replace with ****
	FilterActionWarn   FilterAction = "warn"   // Send warning to player
	FilterActionBlock  FilterAction = "block"  // Block message entirely
	FilterActionLog    FilterAction = "log"    // Just log it for review
)

// SpamDetectionConfig configures spam detection.
type SpamDetectionConfig struct {
	MessagesPerMinute int           `json:"messages_per_minute"` // Threshold for spam
	DuplicateWindow   time.Duration `json:"duplicate_window"`    // Time window for duplicate detection
	Action            FilterAction  `json:"action"`              // What to do when spam detected
}
