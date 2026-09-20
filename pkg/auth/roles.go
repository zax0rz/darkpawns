package auth

// PanelRoleForLevel maps the game's persisted level ladder to the role used
// by the admin login token. OLC zone authorization remains in pkg/olc;
// this helper only assigns the coarse panel role required to reach that gate.
func PanelRoleForLevel(level int) string {
	switch {
	case level >= 40: // LVL_IMPL
		return "admin"
	case level >= 31: // LVL_IMMORT / LVL_BUILDER
		return "builder"
	default:
		return "player"
	}
}
