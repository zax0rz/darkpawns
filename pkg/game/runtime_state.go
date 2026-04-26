package game

// HorseState holds horse mount runtime data.
type HorseState struct {
	CarryWeight int `json:"carry_weight,omitempty"`
	CarryNumber int `json:"carry_number,omitempty"`
	Move        int `json:"move,omitempty"`
	MaxMove     int `json:"max_move,omitempty"`
}

// ObjectRuntimeState replaces CustomData for known object state keys.
type ObjectRuntimeState struct {
	// Corpse/head descriptions
	Name               string `json:"name,omitempty"`
	ShortDesc           string `json:"short_desc,omitempty"`
	LongDesc            string `json:"long_desc,omitempty"`
	ShortDescOverride   string `json:"short_desc_override,omitempty"`

	// Molded objects
	MoldName string `json:"mold_name,omitempty"`
	MoldDesc string `json:"mold_desc,omitempty"`

	// Mail
	MailText string `json:"mail_text,omitempty"`

	// Horse mount
	Horse *HorseState `json:"horse,omitempty"`

	// Escape hatch for genuinely dynamic script state.
	// New Go code should NOT add keys here — add typed fields instead.
	Script map[string]any `json:"script,omitempty"`
}

// MobRuntimeState replaces CustomData for known mob state keys.
type MobRuntimeState struct {
	DamrollBonus int            `json:"damroll_bonus,omitempty"`
	Horse        *HorseState   `json:"horse,omitempty"`
	Script       map[string]any `json:"script,omitempty"`
}
