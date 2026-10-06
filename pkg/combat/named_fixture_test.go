package combat

// Standalone fixtures only; production group payouts use real members.
func NewNamedCombatant(name string, roomVNum int) Combatant {
	return &namedCombatant{name: name, room: roomVNum, isNPC: false}
}

type namedCombatant struct {
	name  string
	room  int
	isNPC bool
}

func (n *namedCombatant) GetName() string                  { return n.name }
func (n *namedCombatant) IsNPC() bool                      { return n.isNPC }
func (n *namedCombatant) GetRoom() int                     { return n.room }
func (n *namedCombatant) GetLevel() int                    { return 0 }
func (n *namedCombatant) GetHP() int                       { return 0 }
func (n *namedCombatant) GetMaxHP() int                    { return 0 }
func (n *namedCombatant) GetAC() int                       { return 0 }
func (n *namedCombatant) GetTHAC0() int                    { return 0 }
func (n *namedCombatant) GetDamageRoll() DiceRoll          { return DiceRoll{} }
func (n *namedCombatant) GetPosition() int                 { return PosStanding }
func (n *namedCombatant) SetPosition(pos int)              {}
func (n *namedCombatant) GetClass() int                    { return 0 }
func (n *namedCombatant) GetStr() int                      { return 0 }
func (n *namedCombatant) GetStrAdd() int                   { return 0 }
func (n *namedCombatant) GetDex() int                      { return 0 }
func (n *namedCombatant) GetInt() int                      { return 0 }
func (n *namedCombatant) GetWis() int                      { return 0 }
func (n *namedCombatant) GetHitroll() int                  { return 0 }
func (n *namedCombatant) GetDamroll() int                  { return 0 }
func (n *namedCombatant) GetSex() int                      { return 1 }
func (n *namedCombatant) GetMaster() string                { return "" }
func (n *namedCombatant) TakeDamage(amount int)            {}
func (n *namedCombatant) Heal(amount int)                  {}
func (n *namedCombatant) SetFightingBody(target Combatant) {}
func (n *namedCombatant) GetFightingBody() Combatant       { return nil }
func (n *namedCombatant) StopFighting()                    {}
func (n *namedCombatant) GetFighting() string              { return "" }
func (n *namedCombatant) SendMessage(msg string)           {}
func (n *namedCombatant) GetSendMessage(msg string)        {}

// defaultDamageRefused is the protection subset visible to the combat
// package alone, used only when no game layer is wired (package tests). It
// emits nothing; production goes through World.DamageRefused.
