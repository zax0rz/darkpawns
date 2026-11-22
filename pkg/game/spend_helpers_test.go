package game

import (
	"strings"
	"testing"
)

// TestClanSpendTreasureNegativeMatchesC pins Clan.SpendTreasure and
// do_clan_bank(CB_WITHDRAW) to C's direct comparison and subtraction at
// src/clan.c:958-963 (DP-1625, R4/R5h):
//   - when treasure < amount with amount <= 0, C refuses;
//   - when treasure >= amount with amount < 0, C still subtracts amount
//     (treasure -= amount, increasing the treasury).
func TestClanSpendTreasureNegativeMatchesC(t *testing.T) {
	t.Run("helper rejects negative balance below negative amount", func(t *testing.T) {
		c := &Clan{Treasure: -100}
		if c.SpendTreasure(-50) {
			t.Fatalf("SpendTreasure(-50) on Treasure=-100 succeeded, want false (C: -100 < -50 at src/clan.c:958)")
		}
		if c.Treasure != -100 {
			t.Fatalf("Treasure = %d, want -100 unchanged", c.Treasure)
		}
	})

	t.Run("helper rejects negative balance on zero amount", func(t *testing.T) {
		c := &Clan{Treasure: -10}
		if c.SpendTreasure(0) {
			t.Fatalf("SpendTreasure(0) on Treasure=-10 succeeded, want false (C: -10 < 0 at src/clan.c:958)")
		}
		if c.Treasure != -10 {
			t.Fatalf("Treasure = %d, want -10 unchanged", c.Treasure)
		}
	})

	t.Run("helper subtracts negative amount when balance suffices", func(t *testing.T) {
		c := &Clan{Treasure: 100}
		if !c.SpendTreasure(-50) {
			t.Fatalf("SpendTreasure(-50) on Treasure=100 failed, want true (C: 100 >= -50)")
		}
		if c.Treasure != 150 {
			t.Fatalf("Treasure = %d, want 150 (C: clan.treasure -= -50 at src/clan.c:963)", c.Treasure)
		}
	})

	t.Run("doClanBank withdraw rejects negative treasury below negative amount", func(t *testing.T) {
		w, p, output := newClanTestWorld(t)
		c := w.Clans.GetClanByIndex(0)
		c.Treasure = -100
		p.SetGold(500)

		// is_number() only accepts digits (src/interpreter.c:1175), so a
		// negative amount reaches src/clan.c:958 via 32-bit atoi() wrap:
		// 4294967246 == 2^32 - 50 -> int32(-50).
		w.doClanBank(p, "4294967246 Wolves", CBWithdraw)

		if !strings.Contains(output.String(), "The clan is not wealthy enough for your needs!") {
			t.Fatalf("output = %q, want C's refusal (src/clan.c:958-960)", output.String())
		}
		if c.TreasureBalance() != -100 {
			t.Fatalf("clan treasure = %d, want -100 unchanged", c.TreasureBalance())
		}
		if p.GetGold() != 500 {
			t.Fatalf("player gold = %d, want 500 unchanged", p.GetGold())
		}
	})

	t.Run("doClanBank withdraw negative amount credits treasury like C", func(t *testing.T) {
		w, p, output := newClanTestWorld(t)
		c := w.Clans.GetClanByIndex(0)
		c.Treasure = 100
		p.SetGold(500)

		// 4294967246 == 2^32 - 50 -> atoi() == -50 (src/clan.c:954).
		w.doClanBank(p, "4294967246 Wolves", CBWithdraw)

		if !strings.Contains(output.String(), "You withdraw from the clan's treasure.") {
			t.Fatalf("output = %q, want C's withdraw ack (src/clan.c:964)", output.String())
		}
		if c.TreasureBalance() != 150 {
			t.Fatalf("clan treasure = %d, want 150 (C: 100 - (-50) at src/clan.c:963)", c.TreasureBalance())
		}
		if p.GetGold() != 450 {
			t.Fatalf("player gold = %d, want 450 (C: 500 + (-50) at src/clan.c:962)", p.GetGold())
		}
	})
}

// TestPlayerSpendGoldNegativeMatchesC pins Player.SpendGold and its reachable
// non-positive-amount callers (do_clan_bank CB_DEPOSIT at src/clan.c:969-974
// and do_clan_apply at src/clan.c:710-715) to C's direct GET_GOLD(ch) < amount
// comparison (DP-1625, R4/R5h).
func TestPlayerSpendGoldNegativeMatchesC(t *testing.T) {
	t.Run("helper rejects negative gold below negative amount", func(t *testing.T) {
		p := NewPlayer(1, "Debtor", 1)
		p.SetGold(-100)
		if p.SpendGold(-50) {
			t.Fatalf("SpendGold(-50) on Gold=-100 succeeded, want false (C: -100 < -50 at src/clan.c:969)")
		}
		if p.GetGold() != -100 {
			t.Fatalf("Gold = %d, want -100 unchanged", p.GetGold())
		}
	})

	t.Run("helper rejects negative gold on zero amount", func(t *testing.T) {
		p := NewPlayer(1, "Debtor", 1)
		p.SetGold(-10)
		if p.SpendGold(0) {
			t.Fatalf("SpendGold(0) on Gold=-10 succeeded, want false (C: -10 < 0 at src/clan.c:710,969)")
		}
		if p.GetGold() != -10 {
			t.Fatalf("Gold = %d, want -10 unchanged", p.GetGold())
		}
	})

	t.Run("helper accepts negative gold at or above negative amount", func(t *testing.T) {
		p := NewPlayer(1, "Debtor", 1)
		p.SetGold(-50)
		if !p.SpendGold(-100) {
			t.Fatalf("SpendGold(-100) on Gold=-50 failed, want true (C: -50 >= -100)")
		}
		if p.GetGold() != 50 {
			t.Fatalf("Gold = %d, want 50 (C: -50 - (-100) at src/clan.c:973)", p.GetGold())
		}
	})

	t.Run("doClanBank deposit rejects mortal with negative gold below negative amount", func(t *testing.T) {
		w, _, output := newClanTestWorld(t)
		c := w.Clans.GetClanByIndex(0)
		c.Treasure = 100

		mortal := NewPlayer(2, "Mortal", 1)
		mortal.Level = 10
		mortal.ClanID = c.ID
		mortal.ClanRank = 1
		mortal.SetGold(-100)
		if err := w.AddPlayer(mortal); err != nil {
			t.Fatalf("AddPlayer: %v", err)
		}
		w.MessageSink = func(name string, msg []byte) {
			if name == mortal.Name {
				output.Write(msg)
			}
		}

		// 4294967246 == 2^32 - 50 -> atoi() == -50 (src/clan.c:954).
		w.doClanBank(mortal, "4294967246", CBDeposit)

		if !strings.Contains(output.String(), "You do not have that kind of money!") {
			t.Fatalf("output = %q, want C's refusal (src/clan.c:969-971)", output.String())
		}
		if mortal.GetGold() != -100 {
			t.Fatalf("mortal gold = %d, want -100 unchanged", mortal.GetGold())
		}
		if c.TreasureBalance() != 100 {
			t.Fatalf("clan treasure = %d, want 100 unchanged", c.TreasureBalance())
		}
	})

	t.Run("doClanApply rejects mortal with negative gold on zero app_fee", func(t *testing.T) {
		w, _, output := newClanTestWorld(t)
		c := w.Clans.GetClanByIndex(0)
		c.AppFee = 0

		applicant := NewPlayer(3, "Applicant", 1)
		applicant.Level = 10
		applicant.SetGold(-10)
		if err := w.AddPlayer(applicant); err != nil {
			t.Fatalf("AddPlayer: %v", err)
		}
		w.MessageSink = func(name string, msg []byte) {
			if name == applicant.Name {
				output.Write(msg)
			}
		}

		w.doClanApply(applicant, "Wolves")

		if !strings.Contains(output.String(), "You cannot afford the application fee!") {
			t.Fatalf("output = %q, want C's refusal (src/clan.c:710-712)", output.String())
		}
		if applicant.ClanID != 0 {
			t.Fatalf("applicant ClanID = %d, want 0 unchanged", applicant.ClanID)
		}
		if applicant.GetGold() != -10 {
			t.Fatalf("applicant gold = %d, want -10 unchanged", applicant.GetGold())
		}
	})
}

// TestMobSpendGoldNegativeMatchesC pins MobInstance.SpendGold to the same
// direct m.Gold < amount comparison as Player.SpendGold (DP-1625, R4/R5h).
func TestMobSpendGoldNegativeMatchesC(t *testing.T) {
	t.Run("rejects negative gold below negative amount", func(t *testing.T) {
		m := &MobInstance{Gold: -100}
		if m.SpendGold(-50) {
			t.Fatalf("SpendGold(-50) on Gold=-100 succeeded, want false")
		}
		if m.GetGold() != -100 {
			t.Fatalf("Gold = %d, want -100 unchanged", m.GetGold())
		}
	})

	t.Run("rejects negative gold on zero amount", func(t *testing.T) {
		m := &MobInstance{Gold: -10}
		if m.SpendGold(0) {
			t.Fatalf("SpendGold(0) on Gold=-10 succeeded, want false")
		}
		if m.GetGold() != -10 {
			t.Fatalf("Gold = %d, want -10 unchanged", m.GetGold())
		}
	})

	t.Run("accepts negative gold at or above negative amount", func(t *testing.T) {
		m := &MobInstance{Gold: -50}
		if !m.SpendGold(-100) {
			t.Fatalf("SpendGold(-100) on Gold=-50 failed, want true")
		}
		if m.GetGold() != 50 {
			t.Fatalf("Gold = %d, want 50", m.GetGold())
		}
	})
}
