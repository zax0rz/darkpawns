package db

// House player-id lookups for the house system's dependency-injected
// get_id_by_name / get_name_by_id pair (game.RegisterHousePlayerLookup).
// C's helpers read the player index file (src/house.c:41-58 via the
// players.ipt index); the port reads the players table with the same
// case-insensitive name matching GetPlayer uses.

// HousePlayerLookups returns the nameByID and idByID pair the game registers
// at startup. An unresolvable id returns "" and an unresolvable name returns
// -1, the values C's helpers return for a miss; the house boot treats those
// as "owner no longer exists".
func (db *DB) HousePlayerLookups() (nameByID func(int64) string, idByName func(string) int64) {
	nameByID = func(id int64) string {
		var name string
		if err := db.queryRow(`SELECT name FROM players WHERE id = ?`, id).Scan(&name); err != nil {
			return ""
		}
		return name
	}
	idByName = func(name string) int64 {
		var id int64
		if err := db.queryRow(`SELECT id FROM players WHERE lower(name) = lower(?)`, name).Scan(&id); err != nil {
			return -1
		}
		return id
	}
	return nameByID, idByName
}
