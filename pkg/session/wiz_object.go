package session

import (
	"fmt"
	"log/slog"
	"strings"
)

func cmdLoad(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Usage: load { obj | mob } <number>\r\n")
		return nil
	}
	kind := strings.ToLower(args[0])
	vnumStr := args[1]
	var vnum int
	if _, err := fmt.Sscanf(vnumStr, "%d", &vnum); err != nil {
		s.Send("That's not a valid number.\r\n")
		return nil
	}
	if vnum < 0 {
		s.Send("A NEGATIVE number??\r\n")
		return nil
	}
	roomVNum := s.player.GetRoom()

	if strings.HasPrefix(kind, "mob") {
		mob, err := s.manager.world.SpawnMob(vnum, roomVNum)
		if err != nil {
			s.Send("There is no monster with that number.\r\n")
			return nil
		}
		slog.Info("(GC) load mob", "who", s.player.Name, "mob", mob.GetShortDesc(), "room", roomVNum)
		s.manager.BroadcastToRoom(roomVNum, []byte(fmt.Sprintf("%s makes a strange magickal gesture.\r\n", s.player.Name)), s.playerName)
		s.manager.BroadcastToRoom(roomVNum, []byte(fmt.Sprintf("%s has created %s!\r\n", s.player.Name, mob.GetShortDesc())), s.playerName)
		s.Send(fmt.Sprintf("You create %s.\r\n", mob.GetShortDesc()))
	} else if strings.HasPrefix(kind, "obj") {
		obj, err := s.manager.world.SpawnObject(vnum, roomVNum)
		if err != nil {
			s.Send("There is no object with that number.\r\n")
			return nil
		}
		slog.Info("(GC) load obj", "who", s.player.Name, "obj", obj.GetShortDesc(), "room", roomVNum)
		s.manager.BroadcastToRoom(roomVNum, []byte(fmt.Sprintf("%s makes a strange magickal gesture.\r\n", s.player.Name)), s.playerName)
		s.manager.BroadcastToRoom(roomVNum, []byte(fmt.Sprintf("%s has created %s!\r\n", s.player.Name, obj.GetShortDesc())), s.playerName)
		s.Send(fmt.Sprintf("You create %s.\r\n", obj.GetShortDesc()))
	} else {
		s.Send("That'll have to be either 'obj' or 'mob'.\r\n")
	}
	return nil
}

// ---------------------------------------------------------------------------
// purge — remove all mobs/objects from room (LVL_IMMORT)
// ---------------------------------------------------------------------------
func cmdPurge(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	roomVNum := s.player.GetRoom()
	if len(args) >= 1 && args[0] != "" {
		// Purge a specific target by name
		targetName := strings.ToLower(strings.Join(args, " "))
		mobs := s.manager.world.GetMobsInRoom(roomVNum)
		for _, mob := range mobs {
			if strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) {
				s.manager.world.ExtractMob(mob)
				s.manager.BroadcastToRoom(roomVNum, []byte(fmt.Sprintf("%s disintegrates %s.\r\n", s.player.Name, mob.GetShortDesc())), s.playerName)
				s.Send("Ok.\r\n")
				slog.Info("(GC) purge", "who", s.player.Name, "target", mob.GetShortDesc())
				return nil
			}
		}
		items := s.manager.world.GetItemsInRoom(roomVNum)
		for _, item := range items {
			if strings.Contains(strings.ToLower(item.GetShortDesc()), targetName) {
				s.manager.world.ExtractObject(item, roomVNum)
				s.manager.BroadcastToRoom(roomVNum, []byte(fmt.Sprintf("%s destroys %s.\r\n", s.player.Name, item.GetShortDesc())), s.playerName)
				s.Send("Ok.\r\n")
				slog.Info("(GC) purge obj", "who", s.player.Name, "target", item.GetShortDesc())
				return nil
			}
		}
		s.Send("Nothing here by that name.\r\n")
		return nil
	}
	// No argument — purge entire room
	s.manager.BroadcastToRoom(roomVNum, []byte(fmt.Sprintf("%s gestures... You are surrounded by scorching flames!\r\n", s.player.Name)), s.playerName)
	for _, mob := range s.manager.world.GetMobsInRoom(roomVNum) {
		s.manager.world.ExtractMob(mob)
	}
	for _, item := range s.manager.world.GetItemsInRoom(roomVNum) {
		s.manager.world.ExtractObject(item, roomVNum)
	}
	s.manager.BroadcastToRoom(roomVNum, []byte("The world seems a little cleaner.\r\n"), s.playerName)
	s.Send("Ok.\r\n")
	return nil
}

// ---------------------------------------------------------------------------
// teleport — teleport a player (LVL_GOD)
// ---------------------------------------------------------------------------
