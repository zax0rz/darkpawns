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
	Name              string `json:"name,omitempty"`
	ShortDesc         string `json:"short_desc,omitempty"`
	LongDesc          string `json:"long_desc,omitempty"`
	ShortDescOverride string `json:"short_desc_override,omitempty"`
	// Keywords for synthetic objects (corpses, money) that have no Prototype.
	// Checked by GetKeywords() before falling back to Prototype.Keywords.
	Keywords string `json:"keywords,omitempty"`

	// Molded objects
	MoldName string `json:"mold_name,omitempty"`
	MoldDesc string `json:"mold_desc,omitempty"`

	// Mail
	MailText string `json:"mail_text,omitempty"`

	// Note — written content from do_write (action_description equivalent).
	// Set when a player writes on an ITEM_NOTE object.
	NoteText string `json:"note_text,omitempty"`

	// Horse mount
	Horse *HorseState `json:"horse,omitempty"`

	// Escape hatch for genuinely dynamic script state.
	// New Go code should NOT add keys here — add typed fields instead.
	Script map[string]any `json:"script,omitempty"`
}

// MobRuntimeState replaces CustomData for known mob state keys.
type MobRuntimeState struct {
	// These overrides preserve do_set() mutations on one live NPC instance;
	// C copies prototype data into char_data, so changing a runtime instance
	// must not rewrite the shared mob prototype.
	AlignmentOverride *int `json:"alignment_override,omitempty"`
	ACOverride        *int `json:"ac_override,omitempty"`
	ClassOverride     *int `json:"class_override,omitempty"`
	SexOverride       *int `json:"sex_override,omitempty"`
	HitrollOverride   *int `json:"hitroll_override,omitempty"`
	StrAddOverride    *int `json:"stradd_override,omitempty"`
	// DamrollOverride is the instance-local GET_DAMROLL value used by C
	// specials that assign points.damroll directly (for example carrion's
	// read_mobile result). A pointer preserves zero as a meaningful override.
	DamrollOverride *int `json:"damroll_override,omitempty"`
	DamrollBonus    int  `json:"damroll_bonus,omitempty"`
	// DamageNumOverride and DamageSidesOverride preserve per-instance
	// mob_specials.damnodice/damsizedice mutations made by native specials.
	// A pointer distinguishes an override of zero from the prototype value.
	DamageNumOverride   *int           `json:"damage_num_override,omitempty"`
	DamageSidesOverride *int           `json:"damage_sides_override,omitempty"`
	Horse               *HorseState    `json:"horse,omitempty"`
	Script              map[string]any `json:"script,omitempty"`
}
