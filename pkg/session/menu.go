package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/pkg/auth"
	"github.com/zax0rz/darkpawns/pkg/game"
	"golang.org/x/crypto/bcrypt"
)

const (
	menuText = "Welcome to Dark Pawns!\r\n" +
		"0) Exit from Dark Pawns.\r\n" +
		"1) Enter the game.\r\n" +
		"2) Enter description.\r\n" +
		"3) Read the background story.\r\n" +
		"4) Change password.\r\n" +
		"5) Delete this character.\r\n\r\n" +
		"   Make your choice: "
	maxMenuPasswordLength = 10
	maxDescriptionLength  = 4096
)

var menuOptions = charOpts(
	"0", "Exit from Dark Pawns",
	"1", "Enter the game",
	"2", "Enter description",
	"3", "Read the background story",
	"4", "Change password",
	"5", "Delete this character",
)

func (s *Session) startReturningMenu(passwordHash string) {
	s.menuActive = true
	s.menuStage = "motd"
	s.menuPasswordHash = passwordHash
	motd := game.ShowMOTD(s.manager.world.WorldPath)
	s.sendCharCreatePrompt("motd", motd+"\r\n\n*** PRESS RETURN: ", nil)
}

func (s *Session) showMainMenu() {
	s.menuActive = true
	s.menuStage = "menu"
	s.sendCharCreatePrompt("menu", menuText, menuOptions)
}

func (s *Session) resendCurrentMenuPrompt() {
	switch s.menuStage {
	case "motd":
		motd := game.ShowMOTD(s.manager.world.WorldPath)
		s.sendCharCreatePrompt("motd", motd+"\r\n\n*** PRESS RETURN: ", nil)
	case "description":
		s.sendCharCreatePrompt("description", "Enter description lines. Type @ or /s to save, /a to abort: ", nil)
	case "password_old":
		s.sendCharCreatePromptWithSecret("menu_password", "Enter your old password: ", nil, true)
	case "password_new":
		s.sendCharCreatePromptWithSecret("menu_password", "Enter a new password: ", nil, true)
	case "password_confirm":
		s.sendCharCreatePromptWithSecret("menu_password", "Please retype password: ", nil, true)
	case "delete_password":
		s.sendCharCreatePromptWithSecret("menu_password", "Enter your password for verification: ", nil, true)
	case "delete_confirm":
		s.sendCharCreatePrompt("delete_confirm", s.deleteConfirmationPrompt(), charOpts("yes", "Permanently delete character"))
	default:
		s.showMainMenu()
	}
}

func (s *Session) handleMenuInput(data json.RawMessage) error {
	var input CharInputData
	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}
	choice := strings.TrimSpace(input.Choice)

	switch s.menuStage {
	case "motd":
		s.showMainMenu()
	case "menu":
		return s.handleMenuChoice(choice)
	case "description":
		s.handleDescriptionLine(input.Choice)
	case "password_old":
		if !s.passwordMatches(choice) {
			s.sendText("\r\nIncorrect password.\r\n")
			s.showMainMenu()
			return nil
		}
		s.menuStage = "password_new"
		s.sendCharCreatePromptWithSecret("menu_password", "Enter a new password: ", nil, true)
	case "password_new":
		if len(choice) < 3 || len(choice) > maxMenuPasswordLength || strings.EqualFold(choice, s.pendingPlayerName()) {
			s.sendCharCreatePromptWithSecret("menu_password", "\r\nIllegal password. Use 3-10 characters and do not use your character name.\r\nPassword: ", nil, true)
			return nil
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(choice), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash new password: %w", err)
		}
		s.menuNewPasswordHash = string(hash)
		s.menuStage = "password_confirm"
		s.sendCharCreatePromptWithSecret("menu_password", "Please retype password: ", nil, true)
	case "password_confirm":
		if bcrypt.CompareHashAndPassword([]byte(s.menuNewPasswordHash), []byte(choice)) != nil {
			s.menuNewPasswordHash = ""
			s.menuStage = "password_new"
			s.sendCharCreatePromptWithSecret("menu_password", "\r\nPasswords don't match... start over.\r\nPassword: ", nil, true)
			return nil
		}
		if err := s.persistChangedPassword(); err != nil {
			return err
		}
		s.sendText("Done.\r\n")
		s.showMainMenu()
	case "delete_password":
		if !s.passwordMatches(choice) {
			s.sendText("\r\nIncorrect password.\r\n")
			s.showMainMenu()
			return nil
		}
		s.menuStage = "delete_confirm"
		s.sendCharCreatePrompt("delete_confirm", s.deleteConfirmationPrompt(), charOpts("yes", "Permanently delete character"))
	case "delete_confirm":
		return s.confirmDelete(choice)
	default:
		s.showMainMenu()
	}
	return nil
}

func (s *Session) handleMenuChoice(choice string) error {
	switch choice {
	case "0":
		s.sendText("Goodbye.\r\n")
		s.menuActive = false
		s.CloseSend()
	case "1":
		if s.player == nil {
			if err := s.completeCharCreation(); err != nil {
				return err
			}
			return nil
		}
		return s.enterReturningPlayer()
	case "2":
		s.menuStage = "description"
		s.menuDescriptionDraft = nil
		current := s.menuDescription
		if s.player != nil {
			current = s.player.Description
		}
		if current != "" {
			s.sendText("Current description:\r\n" + current + "\r\n")
		}
		s.sendCharCreatePrompt("description", "Enter the new text you'd like others to see when they look at you.\r\nType @ or /s to save, /a to abort: ", nil)
	case "3":
		s.sendText(game.ShowBackground(s.manager.world.WorldPath) + "\r\n")
		s.showMainMenu()
	case "4":
		s.menuStage = "password_old"
		s.sendCharCreatePromptWithSecret("menu_password", "Enter your old password: ", nil, true)
	case "5":
		s.menuStage = "delete_password"
		s.sendCharCreatePromptWithSecret("menu_password", "Enter your password for verification: ", nil, true)
	default:
		s.sendText("\r\nThat's not a menu choice!\r\n")
		s.showMainMenu()
	}
	return nil
}

func (s *Session) handleDescriptionLine(line string) {
	command := strings.ToLower(strings.TrimSpace(line))
	if command == "/a" {
		s.menuDescriptionDraft = nil
		s.sendText("Description not changed.\r\n")
		s.showMainMenu()
		return
	}
	if command == "@" || command == "/s" {
		description := strings.Join(s.menuDescriptionDraft, "\r\n")
		s.menuDescriptionDraft = nil
		if s.player == nil {
			s.menuDescription = description
		} else {
			if s.manager.hasDB && s.player.ID > 0 {
				if err := s.manager.db.UpdateDescription(s.player.ID, description); err != nil {
					slog.ErrorContext(s.sessionCtx, "description update failed", s.logAttrs(slog.Any("error", err))...)
					s.sendText("Unable to save description.\r\n")
					s.showMainMenu()
					return
				}
			}
			s.player.Description = description
		}
		s.sendText("Description saved.\r\n")
		s.showMainMenu()
		return
	}

	currentLength := len(strings.Join(s.menuDescriptionDraft, "\r\n"))
	if currentLength+len(line)+2 > maxDescriptionLength {
		s.sendText("Description is too long; type @ or /s to save, /a to abort.\r\n")
		return
	}
	s.menuDescriptionDraft = append(s.menuDescriptionDraft, line)
}

func (s *Session) passwordMatches(password string) bool {
	hash := s.menuPasswordHash
	if s.player == nil {
		hash = s.charPassword
	}
	return hash != "" && bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (s *Session) persistChangedPassword() error {
	if s.player == nil {
		s.charPassword = s.menuNewPasswordHash
	} else if s.manager.hasDB {
		if err := s.manager.db.UpdatePassword(s.player.ID, s.menuNewPasswordHash); err != nil {
			return fmt.Errorf("update password: %w", err)
		}
	}
	s.menuPasswordHash = s.menuNewPasswordHash
	s.menuNewPasswordHash = ""
	return nil
}

func (s *Session) pendingPlayerName() string {
	if s.player != nil {
		return s.player.Name
	}
	return s.charName
}

func (s *Session) deleteConfirmationPrompt() string {
	return fmt.Sprintf("%s:\r\nYOU ARE ABOUT TO DELETE THIS CHARACTER PERMANENTLY.\r\n"+
		"ARE YOU ABSOLUTELY SURE?\r\n\r\nPlease type \"yes\" to confirm: ", s.pendingPlayerName())
}

func (s *Session) confirmDelete(choice string) error {
	if !strings.EqualFold(choice, "yes") {
		s.sendText("\r\nCharacter not deleted.\r\n")
		s.showMainMenu()
		return nil
	}
	if s.player != nil && s.player.GetFlags()&(1<<uint(game.PlrFrozen)) != 0 {
		s.sendText("You try to kill yourself, but the ice stops you.\r\nCharacter not deleted.\r\n")
		s.menuActive = false
		s.CloseSend()
		return nil
	}
	name := s.pendingPlayerName()
	if s.player != nil && s.manager.hasDB && s.player.ID > 0 {
		if err := s.manager.db.DeletePlayer(s.player.ID); err != nil {
			return fmt.Errorf("delete character: %w", err)
		}
	}
	slog.InfoContext(s.sessionCtx, "character self-deleted", s.logAttrs(slog.String("character", name))...)
	s.sendText(fmt.Sprintf("Character '%s' deleted!\r\nGoodbye.\r\n", name))
	s.menuActive = false
	s.player = nil
	s.authenticated = false
	s.CloseSend()
	return nil
}

func (s *Session) enterReturningPlayer() error {
	name := s.player.Name
	grantClassSpells(s.player)
	if err := s.manager.Register(name, s); err != nil {
		return err
	}
	if err := s.manager.world.AddPlayer(s.player); err != nil {
		s.manager.Unregister(name)
		return err
	}

	s.menuActive = false
	s.menuStage = ""
	s.playerName = name
	token, err := auth.GenerateJWT(name, s.isAgent, s.agentKeyID, "")
	if err != nil {
		slog.ErrorContext(s.sessionCtx, "failed to generate JWT token", s.logAttrs(slog.Any("error", err))...)
	}
	s.tokenIssuedAt = time.Now()
	s.sendWelcome(token)
	if s.isAgent || s.wantsStructuredData {
		s.sendFullVarDump()
		if s.isAgent {
			s.SendMemoryBootstrap()
			s.SendMemorySummary()
		}
	}
	enterMsg, err := json.Marshal(ServerMessage{Type: MsgEvent, Data: EventData{Type: "enter", Text: name + " has arrived."}})
	if err == nil {
		s.manager.BroadcastToRoom(s.player.GetRoom(), enterMsg, name)
	}
	if err := ExecuteCommand(s, "look", nil); err != nil {
		slog.ErrorContext(s.sessionCtx, "look command failed on entry", s.logAttrs(slog.Any("error", err))...)
	}
	s.clearMenuState()
	return nil
}

func (s *Session) clearMenuState() {
	s.menuActive = false
	s.menuStage = ""
	s.menuDescription = ""
	s.menuDescriptionDraft = nil
	s.menuPasswordHash = ""
	s.menuNewPasswordHash = ""
}
