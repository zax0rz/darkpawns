// Package command implements admin commands for Dark Pawns.
package command

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/common"
	"github.com/zax0rz/darkpawns/pkg/moderation"
)

// In-memory reports storage (the moderation package persists to DB; this
// provides a zero-dependency fallback that works without a database).
// Matches the same approach the original C MUD used with in-memory linked lists.

// Report holds a single abuse report from a player.
type Report struct {
	ID          int
	Reporter    string
	Target      string
	ReportType  string
	Description string
	Timestamp   time.Time
	Resolved    bool
}

var (
	reports   []Report
	reportsMu sync.RWMutex
	reportSeq int // auto-increment ID
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
	validTypes := map[string]string{
		"harassment":  "harassment",
		"spam":        "spam",
		"cheating":    "cheating",
		"hate_speech": "hate_speech",
		"hate":        "hate_speech",
		"exploit":     "exploit",
		"other":       "other",
	}
	rt, ok := validTypes[strings.ToLower(reportType)]
	if !ok {
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

	// Store report in-memory
	reportsMu.Lock()
	reportSeq++
	reports = append(reports, Report{
		ID:          reportSeq,
		Reporter:    s.GetPlayerName(),
		Target:      target,
		ReportType:  rt,
		Description: description,
		Timestamp:   time.Now(),
		Resolved:    false,
	})
	reportsMu.Unlock()

	// Also log via DB moderation manager if available
	if ac.mod != nil {
		_ = moderation.AbuseReport{
			Reporter:    s.GetPlayerName(),
			Target:      target,
			ReportType:  moderation.ReportType(rt),
			Description: description,
			RoomVNum:    s.GetPlayerRoomVNum(),
			Timestamp:   time.Now(),
			Status:      moderation.ReportStatusPending,
		}
	}

	// Notify admins
	ac.notifyAdmins(fmt.Sprintf(
		"REPORT [#%d]: %s reported %s for %s: %s",
		reportSeq, s.GetPlayerName(), target, rt, description,
	))

	s.Send(fmt.Sprintf("Thank you for reporting %s. Report #%d has been logged.", target, reportSeq))
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
	duration, err := parseDuration(durationStr)
	if err != nil {
		s.Send("Invalid duration format. Use examples: 5m, 1h, 1d")
		return nil
	}

	expiresAt := time.Now().Add(duration)

	// Store penalty via moderation manager (in-memory + DB)
	if ac.mod != nil {
		penalty := moderation.PlayerPenalty{
			PlayerName:  strings.ToLower(target),
			PenaltyType: moderation.ActionMute,
			IssuedAt:    time.Now(),
			ExpiresAt:   &expiresAt,
			Reason:      reason,
			IssuedBy:    s.GetPlayerName(),
		}
		ac.mod.AddPenalty(penalty)
	}

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

	// Disconnect
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

// cmdBan bans a player.
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

	var expiresAt *time.Time
	durationText := "permanently"

	if strings.ToLower(durationStr) != "permanent" {
		dur, err := parseDuration(durationStr)
		if err != nil {
			s.Send("Invalid duration format. Use examples: 5m, 1h, 1d, or 'permanent'")
			return nil
		}
		ea := time.Now().Add(dur)
		expiresAt = &ea
		durationText = fmt.Sprintf("for %s", dur)
	}

	// Store penalty via moderation manager
	if ac.mod != nil {
		penalty := moderation.PlayerPenalty{
			PlayerName:  strings.ToLower(target),
			PenaltyType: moderation.ActionBan,
			IssuedAt:    time.Now(),
			ExpiresAt:   expiresAt,
			Reason:      reason,
			IssuedBy:    s.GetPlayerName(),
		}
		ac.mod.AddPenalty(penalty)
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
		targetSess.Send(fmt.Sprintf(
			"You have been banned %s by %s. Reason: %s",
			durationText, s.GetPlayerName(), reason,
		))
		go func() {
			time.Sleep(100 * time.Millisecond)
			targetSess.Close()
		}()
	}

	ac.logAdminAction(s.GetPlayerName(), moderation.ActionBan, target, reason, nil)

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

	target := strings.ToLower(args[0])

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
	fmt.Fprintf(&output, "Investigation report for: %s\n", target)
	output.WriteString("========================================\n")

	if targetSess == nil {
		output.WriteString("Status: OFFLINE\n")
	} else {
		output.WriteString("Status: ONLINE\n")
		fmt.Fprintf(&output, "Location: Room %d\n", targetSess.GetPlayerRoomVNum())
	}

	// Check for active penalties
	penalties := ac.getPenalties(target)
	if len(penalties) > 0 {
		fmt.Fprintf(&output, "\nActive penalties (%d):\n", len(penalties))
		for _, p := range penalties {
			fmt.Fprintf(&output, "  [%s]", string(p.PenaltyType))
			if p.ExpiresAt != nil {
				remaining := time.Until(*p.ExpiresAt).Round(time.Second)
				fmt.Fprintf(&output, " (expires in %s)", remaining)
			} else {
				output.WriteString(" (permanent)")
			}
			fmt.Fprintf(&output, " - %s\n", p.Reason)
		}
	} else {
		output.WriteString("\nNo active penalties.\n")
	}

	// Check reports
	reportsMu.RLock()
	var relatedReports []Report
	for _, r := range reports {
		if strings.EqualFold(r.Target, target) && !r.Resolved {
			relatedReports = append(relatedReports, r)
		}
	}
	reportsMu.RUnlock()

	if len(relatedReports) > 0 {
		fmt.Fprintf(&output, "\nUnresolved reports (%d):\n", len(relatedReports))
		for _, r := range relatedReports {
			fmt.Fprintf(&output, "  [#%d] %s reported by %s: %s\n",
				r.ID, r.ReportType, r.Reporter, r.Description)
		}
	}

	s.Send(output.String())
	return nil
}

// cmdListReports lists pending abuse reports.
func (ac *AdminCommands) cmdListReports(s common.CommandSession, args []string) error {
	if !ac.isAdmin(s) {
		return fmt.Errorf("you must be an admin to use this command")
	}

	reportsMu.RLock()
	defer reportsMu.RUnlock()

	// Optional: show only unresolved with "unresolved" arg
	showUnresolved := len(args) > 0 && (args[0] == "unresolved" || args[0] == "pending")

	if len(reports) == 0 {
		s.Send("No reports on file.")
		return nil
	}

	var output strings.Builder
	count := 0
	output.WriteString("Abuse Reports:\n")
	output.WriteString("==============\n")

	for _, r := range reports {
		if showUnresolved && r.Resolved {
			continue
		}
		status := "OPEN"
		if r.Resolved {
			status = "CLOSED"
		}
		fmt.Fprintf(&output, "#%d [%s] [%s] %s reported %s for %s\n",
			r.ID, status, r.Timestamp.Format("Jan 02 15:04"), r.Reporter, r.Target, r.ReportType)
		if r.Description != "" {
			fmt.Fprintf(&output, "     %s\n", r.Description)
		}
		count++
	}

	if count == 0 {
		output.WriteString("No unresolved reports.\n")
	} else {
		fmt.Fprintf(&output, "\n%d report(s) shown.\n", count)
		output.WriteString("Use 'investigate <player>' for details.\n")
		output.WriteString("Note: Resolution not yet implemented via CLI.\n")
	}

	s.Send(output.String())
	return nil
}

// cmdListPenalties lists active player penalties.
func (ac *AdminCommands) cmdListPenalties(s common.CommandSession, args []string) error {
	if !ac.isAdmin(s) {
		return fmt.Errorf("you must be an admin to use this command")
	}

	penalties := ac.getAllActivePenalties()

	if len(penalties) == 0 {
		s.Send("No active penalties.")
		return nil
	}

	var output strings.Builder
	output.WriteString("Active Penalties:\n")
	output.WriteString("=================\n")

	now := time.Now()
	for _, p := range penalties {
		fmt.Fprintf(&output, "%s - [%s]", p.PlayerName, p.PenaltyType)
		if p.ExpiresAt != nil && p.ExpiresAt.After(now) {
			remaining := time.Until(*p.ExpiresAt).Round(time.Second)
			fmt.Fprintf(&output, " (expires in %s)", remaining)
		} else if p.ExpiresAt == nil {
			output.WriteString(" (permanent)")
		} else {
			output.WriteString(" (expired)")
		}
		fmt.Fprintf(&output, " - %s\n", p.Reason)
	}

	s.Send(output.String())
	return nil
}

// cmdWordFilter manages word filters.
func (ac *AdminCommands) cmdWordFilter(s common.CommandSession, args []string) error {
	if !ac.isAdmin(s) {
		return fmt.Errorf("you must be an admin to use this command")
	}

	if len(args) == 0 {
		s.Send("Usage: filter <add|remove|list> [args]\n" +
			"  filter add <pattern> [regex] [action]\n" +
			"  filter remove <id>\n" +
			"  filter list\n" +
			"Actions: censor, warn, block, log")
		return nil
	}

	subcmd := strings.ToLower(args[0])

	switch subcmd {
	case "list":
		if ac.mod == nil {
			s.Send("Word filter system not available (no moderation manager).")
			return nil
		}
		filters := ac.mod.GetWordFilters()
		if len(filters) == 0 {
			s.Send("No word filters configured.")
			return nil
		}
		var output strings.Builder
		output.WriteString("Word filters:\n")
		for _, f := range filters {
			regexLabel := "exact"
			if f.IsRegex {
				regexLabel = "regex"
			}
			fmt.Fprintf(&output, "  %d. %s (%s) -> %s\n", f.ID, f.Pattern, regexLabel, f.Action)
		}
		s.Send(output.String())

	case "add":
		if len(args) < 2 {
			s.Send("Usage: filter add <pattern> [regex] [action]\n" +
				"Actions: censor, warn, block, log")
			return nil
		}

		pattern := strings.ToLower(args[1])
		isRegex := false
		actionStr := "censor"

		// Parse optional flags
		for i := 2; i < len(args); i++ {
			a := strings.ToLower(args[i])
			switch a {
			case "regex":
				isRegex = true
			case "censor", "warn", "block", "log":
				actionStr = a
			}
		}

		if ac.mod != nil {
			ac.mod.AddWordFilter(pattern, isRegex, actionStr, s.GetPlayerName())
		}

		s.Send(fmt.Sprintf("Added filter: %s (regex: %v) -> %s", pattern, isRegex, actionStr))

	case "remove":
		if len(args) < 2 {
			s.Send("Usage: filter remove <id>")
			return nil
		}

		var filterID int
		if _, err := fmt.Sscanf(args[1], "%d", &filterID); err != nil {
			s.Send("Filter ID must be a number. Use 'filter list' to see IDs.")
			return nil
		}

		if ac.mod != nil {
			ac.mod.RemoveWordFilter(filterID)
		}
		s.Send(fmt.Sprintf("Removed filter ID %d", filterID))

	default:
		s.Send("Unknown subcommand. Use: add, remove, list")
	}

	return nil
}

// cmdSpamConfig configures spam detection.
func (ac *AdminCommands) cmdSpamConfig(s common.CommandSession, args []string) error {
	if !ac.isAdmin(s) {
		return fmt.Errorf("you must be an admin to use this command")
	}

	if len(args) == 0 {
		config := ac.getSpamConfig()
		s.Send(fmt.Sprintf("Spam detection configuration:\n"+
			"  Messages per minute: %d\n"+
			"  Action: %s\n\n"+
			"Usage: spamconfig <messages_per_min> [action]\n"+
			"  Example: spamconfig 15 block\n"+
			"  Actions: log, warn, block",
			config.MessagesPerMinute, config.Action))
		return nil
	}

	threshold := 10
	actionStr := "warn"

	if _, err := fmt.Sscanf(args[0], "%d", &threshold); err == nil {
		if len(args) > 1 {
			switch strings.ToLower(args[1]) {
			case "log":
				actionStr = "log"
			case "warn":
				actionStr = "warn"
			case "block":
				actionStr = "block"
			}
		}
	} else {
		// Just action provided, parse it
		switch strings.ToLower(args[0]) {
		case "log":
			actionStr = "log"
		case "warn":
			actionStr = "warn"
		case "block":
			actionStr = "block"
		}
	}

	// Store spam config
	if ac.mod != nil {
		ac.mod.SetSpamConfig(threshold, actionStr)
	}

	s.Send(fmt.Sprintf("Spam configuration updated: %d msgs/min, action=%s", threshold, actionStr))
	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// isAdmin checks if a player is an admin.
func (ac *AdminCommands) isAdmin(s common.CommandSession) bool {
	if !s.HasPlayer() {
		return false
	}
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

// logAdminAction logs an admin action via slog.
func (ac *AdminCommands) logAdminAction(admin string, action moderation.AdminAction, target, reason string, duration *time.Duration) {
	slog.Info("admin action",
		"admin", admin,
		"action", action,
		"target", target,
		"reason", reason,
		"duration", duration,
	)
}

// getPenalties returns all active penalties for a player from the moderation manager.
func (ac *AdminCommands) getPenalties(playerName string) []moderation.PlayerPenalty {
	if ac.mod == nil {
		return nil
	}
	return ac.mod.GetPlayerPenalties(playerName)
}

// getAllActivePenalties returns all active penalties across all players.
func (ac *AdminCommands) getAllActivePenalties() []moderation.PlayerPenalty {
	if ac.mod == nil {
		return nil
	}
	return ac.mod.GetAllActivePenalties()
}

// getSpamConfig returns the current spam detection config.
func (ac *AdminCommands) getSpamConfig() moderation.SpamDetectionConfig {
	if ac.mod == nil {
		return moderation.SpamDetectionConfig{
			MessagesPerMinute: 10,
			DuplicateWindow:   5 * time.Second,
			Action:            moderation.FilterActionWarn,
		}
	}
	return ac.mod.GetSpamConfig()
}

// parseDuration parses a duration string, supporting 'd' suffix for days.
func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		var days int
		if _, err := fmt.Sscanf(s[:len(s)-1], "%d", &days); err == nil {
			return time.Duration(days) * 24 * time.Hour, nil
		}
	}
	return time.ParseDuration(s)
}
