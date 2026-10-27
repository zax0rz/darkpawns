package spells

// canReceiveAct mirrors SENDOK for the interface-based fallbacks in this
// package. A live game World handles these messages through game.Act; this
// gate keeps small adapters from delivering act()-equivalent lines to a
// sleeping or writing player.
func canReceiveAct(recipient interface{}) bool {
	type positionReader interface{ GetPosition() int }
	if positioned, ok := recipient.(positionReader); ok && positioned.GetPosition() <= int(PosSleeping) {
		return false
	}
	type flagsReader interface{ GetFlags() uint64 }
	const plrWritingBit = 1 << 4 // PLR_WRITING is bit 4 in src/structs.h:225.
	if flagged, ok := recipient.(flagsReader); ok && flagged.GetFlags()&plrWritingBit != 0 {
		return false
	}
	return true
}
