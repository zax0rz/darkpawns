package spells

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

// savingThrowBeforeKeying is the complete Go table frozen from
// origin/main e54301fb4 before the keyed initializer refactor. Its numeric
// class/category/level positions are deliberately independent of production
// constants and lookup code.
var savingThrowBeforeKeying = [12][5][41]int{
	// CLASS_MAGIC_USER (0)
	{
		// PARA
		{90, 70, 69, 68, 67, 66, 65, 63, 61, 60, 59, 57, 55, 54, 53, 53, 52, 51, 50, 48, 46, 45, 44, 42, 40, 38, 36, 34, 32, 30, 28, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		// ROD
		{90, 55, 53, 51, 49, 47, 45, 43, 41, 40, 39, 37, 35, 33, 31, 30, 29, 27, 25, 23, 21, 20, 19, 17, 15, 14, 13, 12, 11, 10, 9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		// PETRI
		{90, 65, 63, 61, 59, 57, 55, 53, 51, 50, 49, 47, 45, 43, 41, 40, 39, 37, 35, 33, 31, 30, 29, 27, 25, 23, 21, 19, 17, 15, 13, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		// BREATH
		{90, 75, 73, 71, 69, 67, 65, 63, 61, 60, 59, 57, 55, 53, 51, 50, 49, 47, 45, 43, 41, 40, 39, 37, 35, 33, 31, 29, 27, 25, 23, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		// SPELL
		{90, 60, 58, 56, 54, 52, 50, 48, 46, 45, 44, 42, 40, 38, 36, 35, 34, 32, 30, 28, 26, 25, 24, 22, 20, 18, 16, 14, 12, 10, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	},
	// CLASS_CLERIC (1)
	{
		{90, 50, 59, 48, 46, 45, 43, 40, 37, 35, 34, 33, 31, 30, 29, 27, 26, 25, 24, 23, 22, 21, 20, 18, 15, 14, 12, 10, 9, 8, 7, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 70, 69, 68, 66, 65, 63, 60, 57, 55, 54, 53, 51, 50, 49, 47, 46, 45, 44, 43, 42, 41, 40, 38, 35, 34, 32, 30, 29, 28, 27, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 65, 64, 63, 61, 60, 58, 55, 53, 50, 49, 48, 46, 45, 44, 43, 41, 40, 39, 38, 37, 36, 35, 33, 31, 29, 27, 25, 24, 23, 22, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 80, 79, 78, 76, 75, 73, 70, 67, 65, 64, 63, 61, 60, 59, 57, 56, 55, 54, 53, 52, 51, 50, 48, 45, 44, 42, 40, 39, 38, 37, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 75, 74, 73, 71, 70, 68, 65, 63, 60, 59, 58, 56, 55, 54, 53, 51, 50, 49, 48, 47, 46, 45, 43, 41, 39, 37, 35, 34, 33, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	},
	// CLASS_THIEF (2)
	{
		{90, 65, 64, 63, 62, 61, 60, 59, 58, 57, 56, 55, 54, 53, 52, 51, 50, 49, 48, 47, 46, 45, 44, 43, 42, 41, 40, 39, 38, 37, 36, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 70, 68, 66, 64, 62, 60, 58, 56, 54, 52, 50, 48, 46, 44, 42, 40, 38, 36, 34, 32, 30, 28, 26, 24, 22, 20, 18, 16, 14, 13, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 60, 59, 58, 58, 56, 55, 54, 53, 52, 51, 50, 49, 48, 47, 46, 45, 44, 43, 42, 41, 40, 39, 38, 37, 36, 35, 34, 33, 32, 31, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 80, 79, 78, 77, 76, 75, 74, 73, 72, 71, 70, 69, 68, 67, 66, 65, 64, 63, 62, 61, 60, 59, 58, 57, 56, 55, 54, 53, 52, 51, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 75, 73, 71, 69, 67, 65, 63, 61, 59, 57, 55, 53, 51, 49, 47, 45, 43, 41, 39, 37, 35, 33, 31, 29, 27, 25, 23, 21, 19, 17, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	},
	// CLASS_WARRIOR (3)
	{
		{90, 70, 68, 67, 65, 62, 58, 55, 53, 52, 50, 47, 43, 40, 38, 37, 35, 32, 28, 25, 24, 23, 22, 20, 19, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2},
		{90, 80, 78, 77, 75, 72, 68, 65, 63, 62, 60, 57, 53, 50, 48, 47, 45, 42, 38, 35, 34, 33, 32, 30, 29, 27, 26, 25, 24, 23, 22, 20, 18, 16, 14, 12, 10, 8, 6, 5, 4},
		{90, 75, 73, 72, 70, 67, 63, 60, 58, 57, 55, 52, 48, 45, 43, 42, 40, 37, 33, 30, 29, 28, 26, 25, 24, 23, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7},
		{90, 85, 83, 82, 80, 75, 70, 65, 63, 62, 60, 55, 50, 45, 43, 42, 40, 37, 33, 30, 29, 28, 26, 25, 24, 23, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7},
		{90, 85, 83, 82, 80, 77, 73, 70, 68, 67, 65, 62, 58, 55, 53, 52, 50, 47, 43, 40, 39, 38, 36, 35, 34, 33, 31, 30, 29, 28, 27, 25, 23, 21, 19, 17, 15, 13, 11, 9, 7},
	},
	// CLASS_MAGUS (4) — same as Magic User in C
	{
		{90, 70, 69, 68, 67, 66, 65, 63, 61, 60, 59, 57, 55, 54, 53, 53, 52, 51, 50, 48, 46, 45, 44, 42, 40, 38, 36, 34, 32, 30, 28, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 55, 53, 51, 49, 47, 45, 43, 41, 40, 39, 37, 35, 33, 31, 30, 29, 27, 25, 23, 21, 20, 19, 17, 15, 14, 13, 12, 11, 10, 9, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 65, 63, 61, 59, 57, 55, 53, 51, 50, 49, 47, 45, 43, 41, 40, 39, 37, 35, 33, 31, 30, 29, 27, 25, 23, 21, 19, 17, 15, 13, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 75, 73, 71, 69, 67, 65, 63, 61, 60, 59, 57, 55, 53, 51, 50, 49, 47, 45, 43, 41, 40, 39, 37, 35, 33, 31, 29, 27, 25, 23, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 60, 58, 56, 54, 52, 50, 48, 46, 45, 44, 42, 40, 38, 36, 35, 34, 32, 30, 28, 26, 25, 24, 22, 20, 18, 16, 14, 12, 10, 8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	},
	// CLASS_AVATAR (5) — same as Cleric in C
	{
		{90, 50, 59, 48, 46, 45, 43, 40, 37, 35, 34, 33, 31, 30, 29, 27, 26, 25, 24, 23, 22, 21, 20, 18, 15, 14, 12, 10, 9, 8, 7, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 70, 69, 68, 66, 65, 63, 60, 57, 55, 54, 53, 51, 50, 49, 47, 46, 45, 44, 43, 42, 41, 40, 38, 35, 34, 32, 30, 29, 28, 27, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 65, 64, 63, 61, 60, 58, 55, 53, 50, 49, 48, 46, 45, 44, 43, 41, 40, 39, 38, 37, 36, 35, 33, 31, 29, 27, 25, 24, 23, 22, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 80, 79, 78, 76, 75, 73, 70, 67, 65, 64, 63, 61, 60, 59, 57, 56, 55, 54, 53, 52, 51, 50, 48, 45, 44, 42, 40, 39, 38, 37, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 75, 74, 73, 71, 70, 68, 65, 63, 60, 59, 58, 56, 55, 54, 53, 51, 50, 49, 48, 47, 46, 45, 43, 41, 39, 37, 35, 34, 33, 32, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	},
	// CLASS_ASSASSIN (6) — same as Thief in C
	{
		{90, 65, 64, 63, 62, 61, 60, 59, 58, 57, 56, 55, 54, 53, 52, 51, 50, 49, 48, 47, 46, 45, 44, 43, 42, 41, 40, 39, 38, 37, 36, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 70, 68, 66, 64, 62, 60, 58, 56, 54, 52, 50, 48, 46, 44, 42, 40, 38, 36, 34, 32, 30, 28, 26, 24, 22, 20, 18, 16, 14, 13, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 60, 59, 58, 58, 56, 55, 54, 53, 52, 51, 50, 49, 48, 47, 46, 45, 44, 43, 42, 41, 40, 39, 38, 37, 36, 35, 34, 33, 32, 31, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 80, 79, 78, 77, 76, 75, 74, 73, 72, 71, 70, 69, 68, 67, 66, 65, 64, 63, 62, 61, 60, 59, 58, 57, 56, 55, 54, 53, 52, 51, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 75, 73, 71, 69, 67, 65, 63, 61, 59, 57, 55, 53, 51, 49, 47, 45, 43, 41, 39, 37, 35, 33, 31, 29, 27, 25, 23, 21, 19, 17, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	},
	// CLASS_PALADIN (7) — same as Warrior in C
	{
		{90, 70, 68, 67, 65, 62, 58, 55, 53, 52, 50, 47, 43, 40, 38, 37, 35, 32, 28, 25, 24, 23, 22, 20, 19, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2},
		{90, 80, 78, 77, 75, 72, 68, 65, 63, 62, 60, 57, 53, 50, 48, 47, 45, 42, 38, 35, 34, 33, 32, 30, 29, 27, 26, 25, 24, 23, 22, 20, 18, 16, 14, 12, 10, 8, 6, 5, 4},
		{90, 75, 73, 72, 70, 67, 63, 60, 58, 57, 55, 52, 48, 45, 43, 42, 40, 37, 33, 30, 29, 28, 26, 25, 24, 23, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7},
		{90, 85, 83, 82, 80, 75, 70, 65, 63, 62, 60, 55, 50, 45, 43, 42, 40, 37, 33, 30, 29, 28, 26, 25, 24, 23, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7},
		{90, 85, 83, 82, 80, 77, 73, 70, 68, 67, 65, 62, 58, 55, 53, 52, 50, 47, 43, 40, 39, 38, 36, 35, 34, 33, 31, 30, 29, 28, 27, 25, 23, 21, 19, 17, 15, 13, 11, 9, 7},
	},
	// CLASS_NINJA (8) — same as Thief in C
	{
		{90, 65, 64, 63, 62, 61, 60, 59, 58, 57, 56, 55, 54, 53, 52, 51, 50, 49, 48, 47, 46, 45, 44, 43, 42, 41, 40, 39, 38, 37, 36, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 70, 68, 66, 64, 62, 60, 58, 56, 54, 52, 50, 48, 46, 44, 42, 40, 38, 36, 34, 32, 30, 28, 26, 24, 22, 20, 18, 16, 14, 13, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 60, 59, 58, 58, 56, 55, 54, 53, 52, 51, 50, 49, 48, 47, 46, 45, 44, 43, 42, 41, 40, 39, 38, 37, 36, 35, 34, 33, 32, 31, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 80, 79, 78, 77, 76, 75, 74, 73, 72, 71, 70, 69, 68, 67, 66, 65, 64, 63, 62, 61, 60, 59, 58, 57, 56, 55, 54, 53, 52, 51, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 75, 73, 71, 69, 67, 65, 63, 61, 59, 57, 55, 53, 51, 49, 47, 45, 43, 41, 39, 37, 35, 33, 31, 29, 27, 25, 23, 21, 19, 17, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	},
	// CLASS_PSIONIC (9) — same as Thief in C
	{
		{90, 65, 64, 63, 62, 61, 60, 59, 58, 57, 56, 55, 54, 53, 52, 51, 50, 49, 48, 47, 46, 45, 44, 43, 42, 41, 40, 39, 38, 37, 36, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 70, 68, 66, 64, 62, 60, 58, 56, 54, 52, 50, 48, 46, 44, 42, 40, 38, 36, 34, 32, 30, 28, 26, 24, 22, 20, 18, 16, 14, 13, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 60, 59, 58, 58, 56, 55, 54, 53, 52, 51, 50, 49, 48, 47, 46, 45, 44, 43, 42, 41, 40, 39, 38, 37, 36, 35, 34, 33, 32, 31, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 80, 79, 78, 77, 76, 75, 74, 73, 72, 71, 70, 69, 68, 67, 66, 65, 64, 63, 62, 61, 60, 59, 58, 57, 56, 55, 54, 53, 52, 51, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 75, 73, 71, 69, 67, 65, 63, 61, 59, 57, 55, 53, 51, 49, 47, 45, 43, 41, 39, 37, 35, 33, 31, 29, 27, 25, 23, 21, 19, 17, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	},
	// CLASS_RANGER (10) — same as Warrior in C
	{
		{90, 70, 68, 67, 65, 62, 58, 55, 53, 52, 50, 47, 43, 40, 38, 37, 35, 32, 28, 25, 24, 23, 22, 20, 19, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2},
		{90, 80, 78, 77, 75, 72, 68, 65, 63, 62, 60, 57, 53, 50, 48, 47, 45, 42, 38, 35, 34, 33, 32, 30, 29, 27, 26, 25, 24, 23, 22, 20, 18, 16, 14, 12, 10, 8, 6, 5, 4},
		{90, 75, 73, 72, 70, 67, 63, 60, 58, 57, 55, 52, 48, 45, 43, 42, 40, 37, 33, 30, 29, 28, 26, 25, 24, 23, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7},
		{90, 85, 83, 82, 80, 75, 70, 65, 63, 62, 60, 55, 50, 45, 43, 42, 40, 37, 33, 30, 29, 28, 26, 25, 24, 23, 21, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7},
		{90, 85, 83, 82, 80, 77, 73, 70, 68, 67, 65, 62, 58, 55, 53, 52, 50, 47, 43, 40, 39, 38, 36, 35, 34, 33, 31, 30, 29, 28, 27, 25, 23, 21, 19, 17, 15, 13, 11, 9, 7},
	},
	// CLASS_MYSTIC (11) — same as Thief in C
	{
		{90, 65, 64, 63, 62, 61, 60, 59, 58, 57, 56, 55, 54, 53, 52, 51, 50, 49, 48, 47, 46, 45, 44, 43, 42, 41, 40, 39, 38, 37, 36, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 70, 68, 66, 64, 62, 60, 58, 56, 54, 52, 50, 48, 46, 44, 42, 40, 38, 36, 34, 32, 30, 28, 26, 24, 22, 20, 18, 16, 14, 13, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 60, 59, 58, 58, 56, 55, 54, 53, 52, 51, 50, 49, 48, 47, 46, 45, 44, 43, 42, 41, 40, 39, 38, 37, 36, 35, 34, 33, 32, 31, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 80, 79, 78, 77, 76, 75, 74, 73, 72, 71, 70, 69, 68, 67, 66, 65, 64, 63, 62, 61, 60, 59, 58, 57, 56, 55, 54, 53, 52, 51, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		{90, 75, 73, 71, 69, 67, 65, 63, 61, 59, 57, 55, 53, 51, 49, 47, 45, 43, 41, 39, 37, 35, 33, 31, 29, 27, 25, 23, 21, 19, 17, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	},
}

// TestSavingThrowTablePreservesEveryPreKeyingCell compares every stored cell
// after keying with the frozen pre-keying Go table. It includes the level-zero
// sentinel and all stored zero/default cells.
func TestSavingThrowTablePreservesEveryPreKeyingCell(t *testing.T) {
	const expectedCells = 12 * 5 * 41
	cells := 0
	for class := 0; class < 12; class++ {
		for saveType := 0; saveType < 5; saveType++ {
			for level := 0; level < 41; level++ {
				cells++
				if got, want := savingThrowTable[class][saveType][level], savingThrowBeforeKeying[class][saveType][level]; got != want {
					t.Errorf("savingThrowTable[%d][%d][%d] = %d, want frozen pre-keying %d", class, saveType, level, got, want)
				}
			}
		}
	}
	if cells != expectedCells {
		t.Fatalf("checked %d cells, want %d", cells, expectedCells)
	}
}

// TestSavingThrowBeforeKeyingMatchesCFixture keeps the Go-before/Go-after
// preservation claim distinct from the independent C transcribed fixture.
func TestSavingThrowBeforeKeyingMatchesCFixture(t *testing.T) {
	for class := 0; class < 12; class++ {
		for saveType := 0; saveType < 5; saveType++ {
			for level := 0; level < 41; level++ {
				if got, want := savingThrowBeforeKeying[class][saveType][level], savingThrowGolden[class][saveType][level]; got != want {
					t.Errorf("frozen Go table[%d][%d][%d] = %d, want independent C fixture %d", class, saveType, level, got, want)
				}
			}
		}
	}
}

// TestSavingThrowBeforeKeyingChecksum protects the complete frozen baseline
// from accidental edits without consulting the production table.
func TestSavingThrowBeforeKeyingChecksum(t *testing.T) {
	var values []string
	for class := 0; class < 12; class++ {
		for saveType := 0; saveType < 5; saveType++ {
			for level := 0; level < 41; level++ {
				values = append(values, strconv.Itoa(savingThrowBeforeKeying[class][saveType][level]))
			}
		}
	}
	sum := sha256.Sum256([]byte(strings.Join(values, ",")))
	const want = "caf57e8dc021b352253db65c00274194fe1cf1ba393101aa008b1403ad9a6b80"
	if got := fmt.Sprintf("%x", sum[:]); got != want {
		t.Fatalf("frozen pre-keying checksum = %s, want %s", got, want)
	}
}

// TestSavingThrowClassIdentifiersMatchCOrder proves the named production class
// constants retain the C numeric identities independently of the fixture index.
func TestSavingThrowClassIdentifiersMatchCOrder(t *testing.T) {
	tests := []struct {
		name string
		got  int
		want int
	}{
		{"CLASS_MAGIC_USER", combat.ClassMage, 0},
		{"CLASS_CLERIC", combat.ClassCleric, 1},
		{"CLASS_THIEF", combat.ClassThief, 2},
		{"CLASS_WARRIOR", combat.ClassWarrior, 3},
		{"CLASS_MAGUS", combat.ClassMagus, 4},
		{"CLASS_AVATAR", combat.ClassAvatar, 5},
		{"CLASS_ASSASSIN", combat.ClassAssassin, 6},
		{"CLASS_PALADIN", combat.ClassPaladin, 7},
		{"CLASS_NINJA", combat.ClassNinja, 8},
		{"CLASS_PSIONIC", combat.ClassPsionic, 9},
		{"CLASS_RANGER", combat.ClassRanger, 10},
		{"CLASS_MYSTIC", combat.ClassMystic, 11},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %d, want C identity %d", tt.name, tt.got, tt.want)
		}
	}
}

// TestSavingThrowCategoryIdentifiersMatchCOrder proves the named production
// category constants retain SAVING_* numeric identities independently.
func TestSavingThrowCategoryIdentifiersMatchCOrder(t *testing.T) {
	tests := []struct {
		name string
		got  SavingThrowType
		want SavingThrowType
	}{
		{"SAVING_PARA", SaveParalysis, 0},
		{"SAVING_ROD", SaveRodStaff, 1},
		{"SAVING_PETRI", SavePetrify, 2},
		{"SAVING_BREATH", SaveBreath, 3},
		{"SAVING_SPELL", SaveSpell, 4},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %d, want C identity %d", tt.name, tt.got, tt.want)
		}
	}
}

// TestSavingThrowStoredSentinelsAndDefaults checks direct table cells rather
// than GetSavingThrow, whose supported invalid-level path clamps to endpoints.
func TestSavingThrowStoredSentinelsAndDefaults(t *testing.T) {
	for class := 0; class < 12; class++ {
		for saveType := 0; saveType < 5; saveType++ {
			if got := savingThrowTable[class][saveType][0]; got != 90 {
				t.Errorf("savingThrowTable[%d][%d][0] = %d, want stored sentinel 90", class, saveType, got)
			}
			for level := 0; level < 41; level++ {
				if savingThrowBeforeKeying[class][saveType][level] == 0 && savingThrowTable[class][saveType][level] != 0 {
					t.Errorf("savingThrowTable[%d][%d][%d] = %d, want stored zero/default", class, saveType, level, savingThrowTable[class][saveType][level])
				}
			}
		}
	}
}

// TestSavingThrowAccessorEndpointsAndFallbacks covers the full valid endpoints
// and each existing invalid-input fallback without relying on production data.
func TestSavingThrowAccessorEndpointsAndFallbacks(t *testing.T) {
	for class := 0; class < 12; class++ {
		for saveType := 0; saveType < 5; saveType++ {
			for _, level := range []int{0, 40} {
				if got, want := GetSavingThrow(class, level, SavingThrowType(saveType)), savingThrowBeforeKeying[class][saveType][level]; got != want {
					t.Errorf("valid lookup class=%d type=%d level=%d = %d, want %d", class, saveType, level, got, want)
				}
			}
		}
	}
	classCases := []struct {
		class int
		want  int
	}{
		{-1, 0},
		{12, 0},
		{99, 0},
	}
	for _, tt := range classCases {
		if got, want := GetSavingThrow(tt.class, 1, SaveParalysis), savingThrowBeforeKeying[tt.want][0][1]; got != want {
			t.Errorf("invalid class=%d lookup = %d, want class 0 value %d", tt.class, got, want)
		}
	}
	levelCases := []struct {
		level int
		want  int
	}{
		{-1, 0},
		{41, 40},
		{99, 40},
	}
	for _, tt := range levelCases {
		if got, want := GetSavingThrow(3, tt.level, SaveParalysis), savingThrowBeforeKeying[3][0][tt.want]; got != want {
			t.Errorf("invalid level=%d lookup = %d, want clamped level %d value %d", tt.level, got, tt.want, want)
		}
	}
	typeCases := []struct {
		saveType SavingThrowType
	}{
		{-1},
		{5},
		{99},
	}
	for _, tt := range typeCases {
		if got, want := GetSavingThrow(0, 1, tt.saveType), savingThrowBeforeKeying[0][4][1]; got != want {
			t.Errorf("invalid save type=%d lookup = %d, want SaveSpell value %d", tt.saveType, got, want)
		}
	}
}

// TestSavingThrowNPCUsesWarriorTable verifies the existing NPC class
// override while comparing the same seeded draw against a non-NPC Warrior.
func TestSavingThrowNPCUsesWarriorTable(t *testing.T) {
	npc := savingThrowTestCharacter{class: combat.ClassMage, level: 1, npc: true}
	warrior := mockChar{class: combat.ClassWarrior, level: 1}
	dprng.ResetStream(1)
	npcResult := CheckSavingThrow(npc, SaveParalysis)
	dprng.ResetStream(1)
	warriorResult := CheckSavingThrow(warrior, SaveParalysis)
	if npcResult != warriorResult {
		t.Fatalf("NPC saving throw result = %v, want Warrior-table result %v", npcResult, warriorResult)
	}
}

type savingThrowTestCharacter struct {
	class int
	level int
	npc   bool
}

func (c savingThrowTestCharacter) GetClass() int { return c.class }
func (c savingThrowTestCharacter) GetLevel() int { return c.level }
func (c savingThrowTestCharacter) IsNPC() bool   { return c.npc }
