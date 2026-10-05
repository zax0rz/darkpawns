package game

import "time"

// Character saves use process wall time, like C char_to_store. DP_FIXED_TIME
// freezes the calendar only; the accounting seam is independently injectable.
var characterNow = time.Now

func RealNow() time.Time { return characterNow() }

// AccountCharacterSave ports src/db.c:2589-2595. Connection age stays separate.
func (p *Player) AccountCharacterSave() {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := RealNow()
	start := p.playAccountingAt
	if start.IsZero() {
		start = p.ConnectedAt
	}
	if !start.IsZero() && now.After(start) {
		p.PlayedDuration += now.Unix() - start.Unix()
	}
	p.LastLogon = now.Unix()
	p.playAccountingAt = now
}

// ResetPlayAccounting is store_to_char's fresh runtime logon (src/db.c:2436).
func (p *Player) ResetPlayAccounting() {
	p.mu.Lock()
	p.playAccountingAt = RealNow()
	p.mu.Unlock()
}

func (p *Player) PlayingTime() TimeInfoData {
	p.mu.RLock()
	start, played := p.playAccountingAt, p.PlayedDuration
	if start.IsZero() {
		start = p.ConnectedAt
	}
	p.mu.RUnlock()
	if !start.IsZero() {
		played += RealNow().Unix() - start.Unix()
	}
	return TimeInfoData{Day: int(played / SECS_PER_REAL_DAY), Hours: int(played % SECS_PER_REAL_DAY / SECS_PER_REAL_HOUR)}
}

// GetLastLogon reads the save timestamp alongside concurrent saves.
func (p *Player) GetLastLogon() int64 { p.mu.RLock(); defer p.mu.RUnlock(); return p.LastLogon }
