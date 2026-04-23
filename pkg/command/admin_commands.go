// Package command implements admin commands for Dark Pawns.
package command

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/pkg/common"
	"github.com/zax0rz/darkpawns/pkg/moderation"
)

// AdminCommands handles admin/moderation commands.
type AdminCommands struct {
	manager common.CommandManager
	mod     *moderation.Manager
}

// NewAdminCommands creates a new AdminCommands instance.
func NewAdminCommands(manager common.CommandManager, mod *moderation.Manager) *AdminCommands {
	return &AdminCommands{
		manager: manager,
		mod:     mod,
	}
}

// RegisterCommands registers admin commands with the session manager.
func (ac *AdminCommands) RegisterCommands() {
	// Report command for players
	ac.manager.RegisterCommand("report", ac.cmdReport)

	// Admin commands
	ac.manager.RegisterCommand("warn", ac.cmdWarn)
	ac.manager.RegisterCommand("mute", ac.cmdMute)
	ac.manager.RegisterCommand("kick", ac.cmdKick)
	ac.manager.RegisterCommand("ban", ac.cmdBan)
	ac.manager.RegisterCommand("investigate", ac.cmdInvestigate)
	ac.manager.RegisterCommand("reports", ac.cmdListReports)
	ac.manager.RegisterCommand("penalties", ac.cmdListPenalties)
	ac.manager.RegisterCommand("filter", ac.cmdWordFilter)
	ac.manager.RegisterCommand("spamconfig", ac.cmdSpamConfig)
}

// cmdReport allows players to report abusive behavior.
func (ac *AdminCommands) cmdReport(s common.CommandSession, args []string) error {
	if !s.HasPlayer() {
		return fmt.Errorf("you must be logged in to report")
	}

	if len(args) < 2 {
		s.Send("Usage: report <player> <type> [description]\n" +
			"Types: harassment, spam, cheating, hate_speech, exploit, other\n" +
			"Example: report bob harassment \"Being rude in chat\"")
		return nil
	}

	target := args[0]
	reportType := args[1]
	description := ""

	if len(args) > 2 {
		description = strings.Join(args[2:], " ")
	}

	// Validate report type
	var rt moderation.ReportType
	switch strings.ToLower(reportType) {
	case "harassment":
		rt = moderation.ReportTypeHarassment
	case "spam":
		rt = moderation.ReportTypeSpam
	case "cheating":
		rt = moderation.ReportTypeCheating
	case "hate_speech", "hate":
		rt = moderation.ReportTypeHateSpeech
	case "exploit":
		rt = moderation.ReportTypeExploit
	case "other":
		rt = moderation.ReportTypeOther
	default:
		s.Send("Invalid report type. Valid types: harassment, spam, cheating, hate_speech, exploit, other")
		return nil
	}

	// Check if target exists online
	ac.manager.RLock()
	targetExists := false
	for _, sess := range ac.manager.Sessions() {
		if sess.HasPlayer() && strings.EqualFold(sess.GetPlayerName(), target) {
			targetExists = true
			break
		}
	}
	ac.manager.RUnlock()

	if !targetExists {
		s.Send(fmt.Sprintf("Player '%s' is not online.", target))
		return nil
	}

	// Create report (stub - would save to database in real implementation)
	_ = moderation.AbuseReport{
		Reporter:    s.GetPlayerName(),
		Target:      target,
		ReportType:  rt,
		Description: description,
		RoomVNum:    s.GetPlayerRoomVNum(),
		Timestamp:   time.Now(),
		Status:      moderation.ReportStatusPending,
	}

	// In a real implementation, this would save to database
	// For now, just notify admins
	ac.notifyAdmins(fmt.Sprintf(
		"REPORT: %s reported %s for %s: %s",
		s.GetPlayerName(), target, reportType, description,
	))

	s.Send(fmt.Sprintf("Thank you for reporting %s. The admins have been notified.", target))
	return nil
}

// cmdWarn warns a player.
func (ac *AdminCommands) cmdWarn(s common.CommandSession, args []string) error {
	if !ac.isAdmin(s) {
		return fmt.Errorf("you must be an admin to use this command")
	}

	if len(args) < 2 {
		s.Send("Usage: warn <player> <reason>")
		return nil
	}

	target := args[0]
	reason := strings.Join(args[1:], " ")

	// Find target session
	ac.manager.RLock()
	var targetSess common.CommandSession
	for _, sess := range ac.manager.Sessions() {
		if sess.HasPlayer() && strings.EqualFold(sess.GetPlayerName(), target) {
			targetSess = sess
			break
		}
	}
	ac.manager.RUnlock()

	if targetSess == nil {
		s.Send(fmt.Sprintf("Player '%s' is not online.", target))
		return nil
	}

	// Send warning to target
	targetSess.Send(fmt.Sprintf(
		"WARNING from %s: %s\nFurther violations may result in mute, kick, or ban.",
		s.GetPlayerName(), reason,
	))

	// Log action
	ac.logAdminAction(s.GetPlayerName(), moderation.ActionWarn, target, reason, nil)

	s.Send(fmt.Sprintf("Warned %s: %s", target, reason))
	return nil
}

// cmdMute mutes a player for a specified duration.
func (ac *AdminCommands) cmdMute(s common.CommandSession, args []string) error {
	if !ac.isAdmin(s) {
		return fmt.Errorf("you must be an admin to use this command")
	}

	if len(args) < 2 {
		s.Send("Usage: mute <player> <duration> [reason]\n" +
			"Duration examples: 5m, 1h, 1d")
		return nil
	}

	target := args[0]
	durationStr := args[1]
	reason := "No reason given"
	if len(args) > 2 {
		reason = strings.Join(args[2:], " ")
	}

	// Parse duration
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		// Try parsing with day suffix
		if strings.HasSuffix(durationStr, "d") {
			days := durationStr[:len(durationStr)-1]
			var daysInt int
			if _, err := fmt.Sscanf(days, "%d", &daysInt); err == nil {
				duration = time.Duration(daysInt) * 24 * time.Hour
			} else {
				s.Send("Invalid duration format. Use examples: 5m, 1h, 1d")
				return nil
			}
		} else {
			s.Send("Invalid duration format. Use examples: 5m, 1h, 1d")
			return nil
		}
	}

	// Apply mute (stub - would save to database in real implementation)
	expiresAt := time.Now().Add(duration)
	_ = moderation.PlayerPenalty{
		PlayerName:  target,
		PenaltyType: moderation.ActionMute,
		IssuedAt:    time.Now(),
		ExpiresAt:   &expiresAt,
		Reason:      reason,
		IssuedBy:    s.GetPlayerName(),
	}

	// In a real implementation, this would save to database
	// For now, just notify
	ac.notifyPlayer(target, fmt.Sprintf(
		"You have been muted for %s by %s. Reason: %s",
		duration, s.GetPlayerName(), reason,
	))

	ac.logAdminAction(s.GetPlayerName(), moderation.ActionMute, target, reason, &duration)

	s.Send(fmt.Sprintf("Muted %s for %s. Reason: %s", target, duration, reason))
	return nil
}

// cmdKick kicks a player from the game.
func (ac *AdminCommands) cmdKick(s common.CommandSession, args []string) error {
	if !ac.isAdmin(s) {
		return fmt.Errorf("you must be an admin to use this command")
	}

	if len(args) < 2 {
		s.Send("Usage: kick <player> <reason>")
		return nil
	}

	target := args[0]
	reason := strings.Join(args[1:], " ")

	// Find and disconnect target
	ac.manager.RLock()
	var targetSess common.CommandSession
	for _, sess := range ac.manager.Sessions() {
		if sess.HasPlayer() && strings.EqualFold(sess.GetPlayerName(), target) {
			targetSess = sess
			break
		}
	}
	ac.manager.RUnlock()

	if targetSess == nil {
		s.Send(fmt.Sprintf("Player '%s' is not online.", target))
		return nil
	}

	// Notify target
	targetSess.Send(fmt.Sprintf("You have been kicked by %s. Reason: %s", s.GetPlayerName(), reason))

	// Disconnect after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		targetSess.Close()
	}()

	ac.logAdminAction(s.GetPlayerName(), moderation.ActionKick, target, reason, nil)

	s.Send(fmt.Sprintf("Kicked %s. Reason: %s", target, reason))

	// Broadcast to admins
	ac.notifyAdmins(fmt.Sprintf(
		"KICK: %s kicked %s. Reason: %s",
		s.GetPlayerName(), target, reason,
	))

	return nil
}

// cmdBan bans a player (stub - would need database integration).
func (ac *AdminCommands) cmdBan(s common.CommandSession, args []string) error {
	if !ac.isAdmin(s) {
		return fmt.Errorf("you must be an admin to use this command")
	}

	if len(args) < 2 {
		s.Send("Usage: ban <player> <duration> [reason]\n" +
			"Use 'permanent' for permanent ban")
		return nil
	}

	target := args[0]
	durationStr := args[1]
	reason := "No reason given"
	if len(args) > 2 {
		reason = strings.Join(args[2:], " ")
	}

	var duration *time.Duration
	var expiresAt *time.Time

	if strings.ToLower(durationStr) != "permanent" {
		dur, err := time.ParseDuration(durationStr)
		if err != nil {
			// Try parsing with day suffix
			if strings.HasSuffix(durationStr, "d") {
				days := durationStr[:len(durationStr)-1]
				var daysInt int
				if _, err := fmt.Sscanf(days, "%d", &daysInt); err == nil {
					dur = time.Duration(daysInt) * 24 * time.Hour
				} else {
					s.Send("Invalid duration format. Use examples: 5m, 1h, 1d, or 'permanent'")
					return nil
				}
			} else {
				s.Send("Invalid duration format. Use examples: 5m, 1h, 1d, or 'permanent'")
				return nil
			}
		}
		duration = &dur
		ea := time.Now().Add(dur)
		expiresAt = &ea
	}

	// Apply ban (stub - would save to database in real implementation)
	_ = moderation.PlayerPenalty{
		PlayerName:  target,
		PenaltyType: moderation.ActionBan,
		IssuedAt:    time.Now(),
		ExpiresAt:   expiresAt,
		Reason:      reason,
		IssuedBy:    s.GetPlayerName(),
	}

	// Find and disconnect if online
	ac.manager.RLock()
	var targetSess common.CommandSession
	for _, sess := range ac.manager.Sessions() {
		if sess.HasPlayer() && strings.EqualFold(sess.GetPlayerName(), target) {
			targetSess = sess
			break
		}
	}
	ac.manager.RUnlock()

	if targetSess != nil {
		durationText := "permanently"
		if duration != nil {
			durationText = fmt.Sprintf("for %s", *duration)
		}

		targetSess.Send(fmt.Sprintf(
			"You have been banned %s by %s. Reason: %s",
			durationText, s.GetPlayerName(), reason,
		))

		go func() {
			time.Sleep(100 * time.Millisecond)
			targetSess.Close()
		}()
	}

	ac.logAdminAction(s.GetPlayerName(), moderation.ActionBan, target, reason, duration)

	durationText := "permanently"
	if duration != nil {
		durationText = fmt.Sprintf("for %s", *duration)
	}

	s.Send(fmt.Sprintf("Banned %s %s. Reason: %s", target, durationText, reason))

	// Broadcast to admins
	ac.notifyAdmins(fmt.Sprintf(
		"BAN: %s banned %s %s. Reason: %s",
		s.GetPlayerName(), target, durationText, reason,
	))

	return nil
}

// cmdInvestigate shows information about a player.
func (ac *AdminCommands) cmdInvestigate(s common.CommandSession, args []string) error {
	if !ac.isAdmin(s) {
		return fmt.Errorf("you must be an admin to use this command")
	}

	if len(args) == 0 {
		s.Send("Usage: investigate <player>")
		return nil
	}

	target := args[0]

	// Find player session
	ac.manager.RLock()
	var targetSess common.CommandSession
	for _, sess := range ac.manager.Sessions() {
		if sess.HasPlayer() && strings.EqualFold(sess.GetPlayerName(), target) {
			targetSess = sess
			break
		}
	}
	ac.manager.RUnlock()

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Investigation report for: %s\n", target))
	output.WriteString("========================================\n")

	if targetSess == nil {
		output.WriteString("Status: OFFLINE\n")
	} else {
		output.WriteString("Status: ONLINE\n")
		output.WriteString(fmt.Sprintf("Location: Room %d\n", targetSess.GetPlayerRoomVNum()))
		// Note: Player level and health would require additional interface methods
		output.WriteString("Level: [Requires player interface]\n")
		output.WriteString("Health: [Requires player interface]\n")

		// Check for active penalties
		// In a real implementation, this would check the moderation manager
		output.WriteString("\nActive penalties: None (moderation system stub)\n")
	}

	// In a real implementation, this would show report history, etc.
	output.WriteString("\nNote: Full investigation features require database integration.\n")

	s.Send(output.String())
	return nil
}

// cmdListReports lists pending abuse reports (stub).
func (ac *AdminCommands) cmdListReports(s common.CommandSession, args []string) error {
	if !ac.isAdmin(s) {
		return fmt.Errorf("you must be an admin to use this command")
	}

	s.Send("Pending reports:\n" +
		"1. bob reported alice for harassment: \"Being rude\" (Room 8004)\n" +
		"2. charlie reported bob for spam: \"Flooding chat\" (Room 8005)\n\n" +
		"Use 'investigate <player>' for details.\n" +
		"Note: Report system is a stub - requires database integration.")

	return nil
}

// cmdListPenalties lists active player penalties (stub).
func (ac *AdminCommands) cmdListPenalties(s common.CommandSession, args []string) error {
	if !ac.isAdmin(s) {
		return fmt.Errorf("you must be an admin to use this command")
	}

	s.Send("Active penalties:\n" +
		"1. alice - MUTE (expires in 30m) - Reason: Spamming chat\n" +
		"2. bob - WARN - Reason: Harassment\n\n" +
		"Note: Penalty system is a stub - requires database integration.")

	return nil
}

// cmdWordFilter manages word filters (stub).
func (ac *AdminCommands) cmdWordFilter(s common.CommandSession, args []string) error {
	if !ac.isAdmin(s) {
		return fmt.Errorf("you must be an admin to use this command")
	}

	if len(args) == 0 {
		s.Send("Usage: filter <add|remove|list> [args]\n" +
			"  filter add <pattern> [regex] [action]\n" +
			"  filter remove <id>\n" +
			"  filter list")
		return nil
	}

	subcmd := strings.ToLower(args[0])

	switch subcmd {
	case "list":
		s.Send("Word filters:\n" +
			"1. badword (exact) -> censor\n" +
			"2. (?i)hate.*speech (regex) -> block\n\n" +
			"Note: Filter system is a stub - requires database integration.")

	case "add":
		if len(args) < 2 {
			s.Send("Usage: filter add <pattern> [regex] [action]\n" +
				"Actions: censor, warn, block, log")
			return nil
		}

		pattern := args[1]
		isRegex := false
		action := "censor"

		if len(args) > 2 && strings.ToLower(args[2]) == "regex" {
			isRegex = true
			if len(args) > 3 {
				action = args[3]
			}
		} else if len(args) > 2 {
			action = args[2]
		}

		s.Send(fmt.Sprintf("Added filter: %s (regex: %v) -> %s\n"+
			"Note: Filter system is a stub - requires database integration.",
			pattern, isRegex, action))

	case "remove":
		if len(args) < 2 {
			s.Send("Usage: filter remove <id>")
			return nil
		}

		s.Send(fmt.Sprintf("Removed filter ID %s\n"+
			"Note: Filter system is a stub - requires database integration.",
			args[1]))

	default:
		s.Send("Unknown subcommand. Use: add, remove, list")
	}

	return nil
}

// cmdSpamConfig configures spam detection (stub).
func (ac *AdminCommands) cmdSpamConfig(s common.CommandSession, args []string) error {
	if !ac.isAdmin(s) {
		return fmt.Errorf("you must be an admin to use this command")
	}

	if len(args) == 0 {
		s.Send("Spam detection configuration:\n" +
			"  Messages per minute: 10\n" +
			"  Duplicate window: 5s\n" +
			"  Action: warn\n\n" +
			"Usage: spamconfig <threshold> [window] [action]\n" +
			"Example: spamconfig 15 10s block")
		return nil
	}

	s.Send("Spam configuration updated (stub)\n" +
		"Note: Spam detection is a stub - requires full integration.")

	return nil
}

// isAdmin checks if a player is an admin.
// For now, hardcoded - in a real implementation, this would check permissions.
func (ac *AdminCommands) isAdmin(s common.CommandSession) bool {
	if !s.HasPlayer() {
		return false
	}

	// Hardcoded admin list for demo
	admins := map[string]bool{
		"admin":     true,
		"zax0rz":    true,
		"gm":        true,
		"moderator": true,
	}

	return admins[strings.ToLower(s.GetPlayerName())]
}

// notifyAdmins sends a message to all online admins.
func (ac *AdminCommands) notifyAdmins(message string) {
	ac.manager.RLock()
	defer ac.manager.RUnlock()

	for _, sess := range ac.manager.Sessions() {
		if sess.HasPlayer() && ac.isAdmin(sess) {
			sess.Send("[ADMIN] " + message)
		}
	}
}

// notifyPlayer sends a message to a specific player if online.
func (ac *AdminCommands) notifyPlayer(playerName, message string) {
	ac.manager.RLock()
	defer ac.manager.RUnlock()

	for _, sess := range ac.manager.Sessions() {
		if sess.HasPlayer() && strings.EqualFold(sess.GetPlayerName(), playerName) {
			sess.Send(message)
			break
		}
	}
}

// logAdminAction logs an admin action (stub).
func (ac *AdminCommands) logAdminAction(admin string, action moderation.AdminAction, target, reason string, duration *time.Duration) {
	// In a real implementation, this would save to database
	slog.Info("admin action",
		"admin", admin,
		"action", action,
		"target", target,
		"reason", reason,
		"duration", duration,
	)
}
