package admin

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/zax0rz/darkpawns/pkg/audit"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

type entityDraftPathInput struct {
	Kind string `path:"kind" doc:"OLC kind: mob, obj, shop, or zone."`
	VNum string `path:"vnum" doc:"Prototype VNUM or room VNUM for a zone draft."`
}

type entityPatchInput struct {
	Kind string                 `path:"kind"`
	VNum string                 `path:"vnum"`
	Body []entityPatchOperation `json:"-"`
}

type entityPatchOperation struct {
	Kind     string  `json:"kind" doc:"Semantic OLC operation."`
	Text     string  `json:"text,omitempty"`
	Keywords string  `json:"keywords,omitempty"`
	Value    int     `json:"value,omitempty"`
	Float    float64 `json:"float,omitempty"`
	Index    int     `json:"index,omitempty"`
	ToIndex  int     `json:"to_index,omitempty"`
	Bit      int     `json:"bit,omitempty"`
	Enabled  *bool   `json:"enabled,omitempty"`
	Location int     `json:"location,omitempty"`
	Modifier int     `json:"modifier,omitempty"`
	Command  string  `json:"command,omitempty"`
	IfFlag   int     `json:"if_flag,omitempty"`
	Arg1     int     `json:"arg1,omitempty"`
	Arg2     int     `json:"arg2,omitempty"`
	Arg3     int     `json:"arg3,omitempty"`
}

type entityDraftOutput struct {
	Body entityDraftBody
}

type entityDraftBody struct {
	Kind                  string               `json:"kind"`
	VNum                  int                  `json:"vnum"`
	Mob                   *parser.Mob          `json:"mob,omitempty"`
	Object                *parser.Obj          `json:"object,omitempty"`
	Shop                  *parser.ShopProto    `json:"shop,omitempty"`
	Zone                  *entityZoneDraftBody `json:"zone,omitempty"`
	Dirty                 []string             `json:"dirty"`
	LeaseExpiresAt        time.Time            `json:"lease_expires_at"`
	LeaseRemainingSeconds int64                `json:"lease_remaining_seconds"`
}

type entityZoneDraftBody struct {
	Number    int                     `json:"number"`
	Name      string                  `json:"name"`
	TopRoom   int                     `json:"top_room"`
	Lifespan  int                     `json:"lifespan"`
	ResetMode int                     `json:"reset_mode"`
	Commands  []entityZoneCommandView `json:"commands"`
}

type entityZoneCommandView struct {
	Position int    `json:"position"`
	Command  string `json:"command"`
	IfFlag   int    `json:"if_flag"`
	Arg1     int    `json:"arg1"`
	Arg2     int    `json:"arg2"`
	Arg3     int    `json:"arg3"`
}

func registerOLCEntityWrites(api huma.API, world *game.World, database *db.DB, writes OLCWriteStateProvider, presence OLCPresenceProvider, auditLogger *audit.AuditLogger, drafts *olc.DraftStore) {
	gate := olcGate(world, database)

	register := func(method, path, operationID, summary, description string, handler func(context.Context, *entityDraftPathInput) (*entityDraftOutput, error)) {
		huma.Register(api, huma.Operation{
			OperationID: operationID,
			Method:      method,
			Path:        path,
			Summary:     summary,
			Description: description,
			Middlewares: huma.Middlewares{gate},
		}, handler)
	}
	register(http.MethodPost, "/admin/olc/{kind}/{vnum}", "open-olc-entity-draft", "Open an OLC draft", "Claims a mob, object, shop, or room-scoped zone lease and recovers its draft.", func(ctx context.Context, in *entityDraftPathInput) (*entityDraftOutput, error) {
		return openEntityDraft(ctx, in, world, writes, presence, drafts)
	})
	register(http.MethodGet, "/admin/olc/{kind}/{vnum}/draft", "get-olc-entity-draft", "Get an OLC draft", "Returns the effective typed working copy and lease state.", func(ctx context.Context, in *entityDraftPathInput) (*entityDraftOutput, error) {
		return getEntityDraft(ctx, in, writes, drafts)
	})
	register(http.MethodDelete, "/admin/olc/{kind}/{vnum}/draft", "discard-olc-entity-draft", "Discard an OLC draft", "Discards the typed draft and releases its lease.", func(ctx context.Context, in *entityDraftPathInput) (*entityDraftOutput, error) {
		_, err := discardEntityDraft(ctx, in, writes, presence, drafts)
		return &entityDraftOutput{}, err
	})

	huma.Register(api, huma.Operation{
		OperationID: "patch-olc-entity-draft",
		Method:      http.MethodPatch,
		Path:        "/admin/olc/{kind}/{vnum}/draft",
		Summary:     "Patch an OLC draft",
		Description: "Applies an ordered typed operation list to a mob, object, shop, or zone draft.",
		Middlewares: huma.Middlewares{gate},
	}, func(ctx context.Context, in *entityPatchInput) (*entityDraftOutput, error) {
		return patchEntityDraft(ctx, in, world, writes, drafts)
	})
	huma.Register(api, huma.Operation{
		OperationID: "commit-olc-entity-draft",
		Method:      http.MethodPost,
		Path:        "/admin/olc/{kind}/{vnum}/draft/commit",
		Summary:     "Commit an OLC draft",
		Description: "Commits the effective typed working copy to memory, marks the zone dirty, audits it, and releases the lease.",
		Middlewares: huma.Middlewares{gate},
	}, func(ctx context.Context, in *entityDraftPathInput) (*entityDraftOutput, error) {
		return commitEntityDraft(ctx, in, world, writes, presence, auditLogger, drafts)
	})
}

func openEntityDraft(ctx context.Context, in *entityDraftPathInput, world *game.World, writes OLCWriteStateProvider, presence OLCPresenceProvider, drafts *olc.DraftStore) (*entityDraftOutput, error) {
	kind, vnum, err := entityPath(in)
	if err != nil {
		return nil, err
	}
	owner, err := webOwner(ctx)
	if err != nil {
		return nil, err
	}
	if writes == nil || drafts == nil {
		return nil, huma.NewError(http.StatusServiceUnavailable, "OLC state unavailable")
	}
	snapshot, err := snapshotEntity(world, kind, vnum)
	if err != nil {
		return nil, err
	}
	hadClaim := hasOLCClaim(writes, kind, vnum, owner)
	if holder, claimed := writes.ClaimOLC(kind, vnum, owner, olc.DefaultClaimTTL); !claimed {
		return nil, entityClaimConflict(writes, kind, vnum, holder)
	}
	draft, err := drafts.OpenEntity(owner.Identity(), kind, vnum, snapshot)
	if err != nil {
		if !hadClaim {
			writes.ReleaseOLC(kind, vnum, owner)
		}
		return nil, huma.NewError(http.StatusConflict, err.Error())
	}
	if !hadClaim && presence != nil {
		presence.EmitOLCPresence(owner.DisplayName(), true)
	}
	return &entityDraftOutput{Body: makeEntityDraftBody(draft, claimEntry(writes, kind, vnum))}, nil
}

func getEntityDraft(ctx context.Context, in *entityDraftPathInput, writes OLCWriteStateProvider, drafts *olc.DraftStore) (*entityDraftOutput, error) {
	kind, vnum, err := entityPath(in)
	if err != nil {
		return nil, err
	}
	owner, err := webOwner(ctx)
	if err != nil {
		return nil, err
	}
	if writes == nil || drafts == nil {
		return nil, huma.NewError(http.StatusServiceUnavailable, "OLC state unavailable")
	}
	if !writes.RenewOLC(kind, vnum, owner, olc.DefaultClaimTTL) {
		return nil, entityLeaseExpired(writes, kind, vnum)
	}
	draft, ok := drafts.GetEntity(owner.Identity())
	if !ok || draft.Kind != kind || draft.VNum != vnum {
		return nil, huma.NewError(http.StatusNotFound, "OLC draft not found")
	}
	return &entityDraftOutput{Body: makeEntityDraftBody(draft, claimEntry(writes, kind, vnum))}, nil
}

func patchEntityDraft(ctx context.Context, in *entityPatchInput, world *game.World, writes OLCWriteStateProvider, drafts *olc.DraftStore) (*entityDraftOutput, error) {
	kind, vnum, err := entityPath(&entityDraftPathInput{Kind: in.Kind, VNum: in.VNum})
	if err != nil {
		return nil, err
	}
	owner, err := webOwner(ctx)
	if err != nil {
		return nil, err
	}
	if writes == nil || drafts == nil {
		return nil, huma.NewError(http.StatusServiceUnavailable, "OLC state unavailable")
	}
	if !writes.RenewOLC(kind, vnum, owner, olc.DefaultClaimTTL) {
		return nil, entityLeaseExpired(writes, kind, vnum)
	}
	operations := make([]olc.Operation, 0, len(in.Body))
	for i := range in.Body {
		input := in.Body[i]
		operation, err := entityOperation(kind, input)
		if err != nil {
			return nil, err
		}
		if err := validateZoneCommandPrototypes(world, operation); err != nil {
			return nil, err
		}
		bindEntityLiveScriptOperation(world, kind, vnum, &operation)
		operations = append(operations, operation)
	}
	draft, err := drafts.PatchEntity(owner.Identity(), operations)
	if err != nil {
		return nil, huma.NewError(http.StatusBadRequest, err.Error())
	}
	return &entityDraftOutput{Body: makeEntityDraftBody(draft, claimEntry(writes, kind, vnum))}, nil
}

// validateZoneCommandPrototypes is zedit's prototype check (zedit.c:1039-1210):
// every prototype argument must resolve through real_mobile/real_object, or
// C re-prompts "That mobile/object does not exist". C has no vnum-0 rule; 0
// is accepted exactly when a prototype 0 exists.
func validateZoneCommandPrototypes(world *game.World, op olc.Operation) error {
	if (op.Kind != olc.OpAddZoneCommand && op.Kind != olc.OpModifyZoneCommand) || op.Command == nil || world == nil {
		return nil
	}
	mob := func(vnum int) error {
		if _, ok := world.GetMobPrototype(vnum); !ok {
			return &webOLCError{Message: fmt.Sprintf("That mobile does not exist (vnum %d).", vnum)}
		}
		return nil
	}
	obj := func(vnum int) error {
		if _, ok := world.GetObjPrototype(vnum); !ok {
			return &webOLCError{Message: fmt.Sprintf("That object does not exist (vnum %d).", vnum)}
		}
		return nil
	}
	c := op.Command
	switch c.Command {
	case "M":
		return mob(c.Arg1)
	case "O", "E", "G":
		return obj(c.Arg1)
	case "P":
		if err := obj(c.Arg1); err != nil {
			return err
		}
		return obj(c.Arg3)
	case "R":
		if c.Arg2 != 0 {
			return obj(c.Arg3)
		}
		return mob(c.Arg3)
	}
	return nil
}

func bindEntityLiveScriptOperation(world *game.World, kind olc.Kind, vnum int, op *olc.Operation) {
	switch op.Kind {
	case olc.OpSetMobScriptName:
		op.SetScriptName = func(name string) bool {
			mob, ok := world.SnapshotMob(vnum)
			return ok && world.SetMobScript(vnum, name, mob.LuaFunctions)
		}
	case olc.OpSetMobScriptFlag:
		op.SetScriptFlag = func(bit int, enabled bool) bool {
			mob, ok := world.SnapshotMob(vnum)
			if !ok {
				return false
			}
			flags := mob.LuaFunctions
			if enabled {
				flags |= 1 << uint(bit)
			} else {
				flags &^= 1 << uint(bit)
			}
			return world.SetMobScript(vnum, mob.ScriptName, flags)
		}
	case olc.OpSetObjScriptName:
		op.SetScriptName = func(name string) bool {
			object, ok := world.SnapshotObj(vnum)
			return ok && world.SetObjScript(vnum, name, object.LuaFunctions)
		}
	case olc.OpSetObjScriptFlag:
		op.SetScriptFlag = func(bit int, enabled bool) bool {
			object, ok := world.SnapshotObj(vnum)
			if !ok {
				return false
			}
			flags := object.LuaFunctions
			if enabled {
				flags |= 1 << uint(bit)
			} else {
				flags &^= 1 << uint(bit)
			}
			return world.SetObjScript(vnum, object.ScriptName, flags)
		}
	default:
		return
	}
}

func commitEntityDraft(ctx context.Context, in *entityDraftPathInput, world *game.World, writes OLCWriteStateProvider, presence OLCPresenceProvider, auditLogger *audit.AuditLogger, drafts *olc.DraftStore) (*entityDraftOutput, error) {
	kind, vnum, err := entityPath(in)
	if err != nil {
		return nil, err
	}
	owner, err := webOwner(ctx)
	if err != nil {
		return nil, err
	}
	if writes == nil || drafts == nil {
		return nil, huma.NewError(http.StatusServiceUnavailable, "OLC state unavailable")
	}
	var committed olc.EntityDraft
	committed, err = drafts.CommitEntityWith(owner.Identity(), func(draft olc.EntityDraft) error {
		if draft.Kind != kind || draft.VNum != vnum {
			return huma.NewError(http.StatusConflict, "draft does not match resource")
		}
		if !writes.RenewOLC(kind, vnum, owner, olc.DefaultClaimTTL) {
			return entityLeaseExpired(writes, kind, vnum)
		}
		zone, ok := entityZone(world, kind, vnum)
		if !ok {
			return huma.NewError(http.StatusNotFound, "no zone covers VNUM")
		}
		lock := olc.ZoneSaveLock(zone.Number)
		lock.Lock()
		defer lock.Unlock()
		success := false
		switch kind {
		case olc.KindMob:
			success = olc.CommitMob(olc.MobCommitInput{Draft: draft, Actor: owner.DisplayName(), IPAddress: clientIPFrom(ctx), Commit: world.CommitEditedMob, LiveScript: func() (parser.Mob, bool) { return world.SnapshotMob(vnum) }, MarkDirty: func() { writes.MarkOLCDirty(kind, zone.Number) }, Audit: auditAdapter(auditLogger)})
		case olc.KindObject:
			success = olc.CommitObj(olc.ObjectCommitInput{Draft: draft, Actor: owner.DisplayName(), IPAddress: clientIPFrom(ctx), Commit: world.CommitEditedObj, LiveScript: func() (parser.Obj, bool) { return world.SnapshotObj(vnum) }, MarkDirty: func() { writes.MarkOLCDirty(kind, zone.Number) }, Audit: auditAdapter(auditLogger)})
		case olc.KindShop:
			success = olc.CommitShop(olc.ShopCommitInput{Draft: draft, Actor: owner.DisplayName(), IPAddress: clientIPFrom(ctx), Commit: world.CommitEditedShop, MarkDirty: func() { writes.MarkOLCDirty(kind, zone.Number) }, Audit: auditAdapter(auditLogger)})
		case olc.KindZone:
			success = olc.CommitZone(olc.ZoneCommitInput{Draft: draft, Actor: owner.DisplayName(), IPAddress: clientIPFrom(ctx), Commit: func(room int, working parser.Zone) bool { return world.CommitEditedZone(zone.Number, room, working) }, MarkDirty: func() { writes.MarkOLCDirty(kind, zone.Number) }, Audit: auditAdapter(auditLogger)})
		}
		if !success {
			return huma.NewError(http.StatusConflict, "OLC commit failed")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	writes.ReleaseOLC(kind, vnum, owner)
	if presence != nil {
		presence.EmitOLCPresence(owner.DisplayName(), false)
	}
	return &entityDraftOutput{Body: makeEntityDraftBody(committed, olc.ClaimEntry{})}, nil
}

func discardEntityDraft(ctx context.Context, in *entityDraftPathInput, writes OLCWriteStateProvider, presence OLCPresenceProvider, drafts *olc.DraftStore) (*struct{}, error) {
	kind, vnum, err := entityPath(in)
	if err != nil {
		return nil, err
	}
	owner, err := webOwner(ctx)
	if err != nil {
		return nil, err
	}
	if writes == nil || drafts == nil {
		return nil, huma.NewError(http.StatusServiceUnavailable, "OLC state unavailable")
	}
	draft, ok := drafts.GetEntity(owner.Identity())
	if !ok || draft.Kind != kind || draft.VNum != vnum {
		return nil, huma.NewError(http.StatusNotFound, "OLC draft not found")
	}
	drafts.DiscardEntity(owner.Identity())
	writes.ReleaseOLC(kind, vnum, owner)
	if presence != nil {
		presence.EmitOLCPresence(owner.DisplayName(), false)
	}
	return &struct{}{}, nil
}

func entityPath(in *entityDraftPathInput) (olc.Kind, int, error) {
	kind, ok := olcKindForPath(in.Kind)
	if !ok || kind == olc.KindRoom {
		return "", 0, huma.NewError(http.StatusBadRequest, "unknown OLC entity kind")
	}
	vnum, err := strconv.Atoi(in.VNum)
	if err != nil {
		return "", 0, huma.NewError(http.StatusBadRequest, "invalid vnum")
	}
	return kind, vnum, nil
}

func snapshotEntity(world *game.World, kind olc.Kind, vnum int) (olc.EntityValue, error) {
	zone, ok := entityZone(world, kind, vnum)
	if !ok {
		return olc.EntityValue{}, huma.NewError(http.StatusNotFound, "no zone covers VNUM")
	}
	value := olc.EntityValue{}
	switch kind {
	case olc.KindMob:
		mob, exists := world.SnapshotMob(vnum)
		if !exists {
			mob = newOLCMob(vnum)
		}
		value.Mob = mob
	case olc.KindObject:
		object, exists := world.SnapshotObj(vnum)
		if !exists {
			object = newOLCObject(vnum)
		}
		value.Object = object
	case olc.KindShop:
		shop, exists := world.SnapshotShop(vnum)
		if !exists {
			value.Shop = newOLCShop(vnum)
			break
		}
		value.Shop = shopToProto(shop)
	case olc.KindZone:
		zoneCopy, exists := world.SnapshotZone(zone.Number)
		if !exists {
			return value, huma.NewError(http.StatusNotFound, "zone not found")
		}
		roomVNum := vnum
		if vnum != zone.Number {
			zoneCopy.Commands = olc.ZoneCommandsForRoom(zoneCopy.Commands, vnum)
		} else {
			roomVNum = 0
		}
		value.Zone = olc.ZoneDraft{Zone: zoneCopy, RoomVNum: roomVNum}
	}
	return value, nil
}

func entityOperation(kind olc.Kind, input entityPatchOperation) (olc.Operation, error) {
	op := olc.Operation{Text: input.Text, Float: input.Float, Value: input.Value, Bit: input.Bit, Index: input.Index, ToIndex: input.ToIndex}
	if input.Enabled != nil {
		if *input.Enabled {
			op.Value = 1
		} else {
			op.Value = 0
		}
	}
	switch kind {
	case olc.KindMob:
		switch input.Kind {
		case "set_keywords":
			op.Kind = olc.OpSetMobKeywords
		case "set_short_description":
			op.Kind = olc.OpSetMobShortDescription
		case "set_long_description":
			op.Kind = olc.OpSetMobLongDescription
		case "set_detailed_description":
			op.Kind = olc.OpSetMobDetailedDescription
		case "set_sex":
			op.Kind = olc.OpSetMobSex
		case "set_hitroll":
			op.Kind = olc.OpSetMobHitroll
		case "set_damroll":
			op.Kind = olc.OpSetMobDamroll
		case "set_damage_dice":
			op.Kind = olc.OpSetMobNumDamageDice
		case "set_damage_sides":
			op.Kind = olc.OpSetMobSizeDamageDice
		case "set_hp_dice":
			op.Kind = olc.OpSetMobNumHPDice
		case "set_hp_sides":
			op.Kind = olc.OpSetMobSizeHPDice
		case "set_hp_plus":
			op.Kind = olc.OpSetMobAddHP
		case "set_ac":
			op.Kind = olc.OpSetMobAC
		case "set_exp":
			op.Kind = olc.OpSetMobExp
		case "set_gold":
			op.Kind = olc.OpSetMobGold
		case "set_position":
			op.Kind = olc.OpSetMobPosition
		case "set_default_position":
			op.Kind = olc.OpSetMobDefaultPosition
		case "set_attack":
			op.Kind = olc.OpSetMobAttack
		case "set_level":
			op.Kind = olc.OpSetMobLevel
		case "set_alignment":
			op.Kind = olc.OpSetMobAlignment
		case "set_race":
			op.Kind = olc.OpSetMobRace
		case "set_action_flag":
			op.Kind = olc.OpSetMobActionFlag
		case "set_affect_flag":
			op.Kind = olc.OpSetMobAffectFlag
		case "set_noise":
			op.Kind = olc.OpSetMobNoise
		case "set_script_name":
			op.Kind = olc.OpSetMobScriptName
		case "set_script_flag":
			op.Kind = olc.OpSetMobScriptFlag
		default:
			return olc.Operation{}, &webOLCError{Message: fmt.Sprintf("unknown mob operation %q", input.Kind)}
		}
	case olc.KindObject:
		switch input.Kind {
		case "set_keywords":
			op.Kind = olc.OpSetObjKeywords
		case "set_short_description":
			op.Kind = olc.OpSetObjShortDescription
		case "set_long_description":
			op.Kind = olc.OpSetObjLongDescription
		case "set_action_description":
			op.Kind = olc.OpSetObjActionDescription
		case "set_value1":
			op.Kind = olc.OpSetObjValue1
		case "set_value2":
			op.Kind = olc.OpSetObjValue2
		case "set_value3":
			op.Kind = olc.OpSetObjValue3
		case "set_value4":
			op.Kind = olc.OpSetObjValue4
		case "toggle_container_flag":
			op.Kind = olc.OpToggleObjContainerFlag
		case "set_type":
			op.Kind = olc.OpSetObjType
		case "set_extra_flag":
			op.Kind = olc.OpSetObjExtraFlag
		case "set_wear_flag":
			op.Kind = olc.OpSetObjWearFlag
		case "set_weight":
			op.Kind = olc.OpSetObjWeight
		case "set_cost":
			op.Kind = olc.OpSetObjCost
		case "set_cost_per_day":
			op.Kind = olc.OpSetObjCostPerDay
		case "set_timer":
			op.Kind = olc.OpSetObjTimer
		case "set_level":
			op.Kind = olc.OpSetObjLevel
		case "set_script_name":
			op.Kind = olc.OpSetObjScriptName
		case "set_script_flag":
			op.Kind = olc.OpSetObjScriptFlag
		case "add_affect":
			op.Kind, op.Affect = olc.OpAddObjAffect, &parser.ObjAffect{Location: input.Location, Modifier: input.Modifier}
		case "remove_affect":
			op.Kind = olc.OpRemoveObjAffect
		case "add_extra_description":
			op.Kind, op.Extra = olc.OpAddObjExtraDescription, &parser.ExtraDesc{Keywords: input.Keywords, Description: input.Text}
		case "remove_extra_description":
			op.Kind = olc.OpRemoveObjExtraDescription
		case "set_extra_keywords":
			op.Kind = olc.OpSetObjExtraKeywords
		case "set_extra_description":
			op.Kind = olc.OpSetObjExtraDescription
		default:
			return olc.Operation{}, &webOLCError{Message: fmt.Sprintf("unknown object operation %q", input.Kind)}
		}
	case olc.KindShop:
		switch input.Kind {
		case "add_product":
			op.Kind = olc.OpAddShopProduct
		case "remove_product":
			op.Kind = olc.OpRemoveShopProduct
		case "set_buy_profit":
			op.Kind = olc.OpSetShopBuyProfit
			op.Value = int(input.Float * 100)
		case "set_sell_profit":
			op.Kind = olc.OpSetShopSellProfit
			op.Value = int(input.Float * 100)
		case "set_keeper":
			op.Kind = olc.OpSetShopKeeper
		case "set_flags":
			op.Kind = olc.OpSetShopFlags
		case "set_with_who":
			op.Kind = olc.OpSetShopWithWho
		case "add_room":
			op.Kind = olc.OpAddShopRoom
		case "remove_room":
			op.Kind = olc.OpRemoveShopRoom
		case "set_open_hour1":
			op.Kind = olc.OpSetShopOpenHour1
		case "set_open_hour2":
			op.Kind = olc.OpSetShopOpenHour2
		case "set_close_hour1":
			op.Kind = olc.OpSetShopCloseHour1
		case "set_close_hour2":
			op.Kind = olc.OpSetShopCloseHour2
		case "set_no_item1":
			op.Kind, op.Index = olc.OpSetShopMessage, 0
		case "set_no_item2":
			op.Kind, op.Index = olc.OpSetShopMessage, 1
		case "set_no_buy":
			op.Kind, op.Index = olc.OpSetShopMessage, 2
		case "set_no_cash1":
			op.Kind, op.Index = olc.OpSetShopMessage, 3
		case "set_no_cash2":
			op.Kind, op.Index = olc.OpSetShopMessage, 4
		case "set_buy_message":
			op.Kind, op.Index = olc.OpSetShopMessage, 5
		case "set_sell_message":
			op.Kind, op.Index = olc.OpSetShopMessage, 6
		case "add_namelist":
			op.Kind = olc.OpAddShopNamelist
			op.Text = input.Text
			if input.Text == "" {
				op.Text = input.Keywords
			}
		case "remove_namelist":
			op.Kind = olc.OpRemoveShopNamelist
		case "set_no_trade":
			op.Kind = olc.OpSetShopNoTrade
		default:
			return olc.Operation{}, &webOLCError{Message: fmt.Sprintf("unknown shop operation %q", input.Kind)}
		}
	case olc.KindZone:
		switch input.Kind {
		case "set_name":
			op.Kind = olc.OpSetZoneName
		case "set_lifespan":
			op.Kind = olc.OpSetZoneLifespan
		case "set_reset_mode":
			op.Kind = olc.OpSetZoneResetMode
		case "set_top_room":
			op.Kind = olc.OpSetZoneTopRoom
			op.Low, op.High = 0, 99999
		case "add_command":
			op.Kind = olc.OpAddZoneCommand
			op.Command = &parser.ZoneCommand{Command: input.Command, IfFlag: input.IfFlag, Arg1: input.Arg1, Arg2: input.Arg2, Arg3: input.Arg3}
		case "modify_command":
			op.Kind = olc.OpModifyZoneCommand
			op.Command = &parser.ZoneCommand{Command: input.Command, IfFlag: input.IfFlag, Arg1: input.Arg1, Arg2: input.Arg2, Arg3: input.Arg3}
		case "remove_command":
			op.Kind = olc.OpRemoveZoneCommand
		case "reorder_command":
			op.Kind = olc.OpReorderZoneCommand
		default:
			return olc.Operation{}, &webOLCError{Message: fmt.Sprintf("unknown zone operation %q", input.Kind)}
		}
	default:
		return olc.Operation{}, huma.NewError(http.StatusBadRequest, "unsupported OLC entity kind")
	}
	return op, nil
}

func makeEntityDraftBody(draft olc.EntityDraft, entry olc.ClaimEntry) entityDraftBody {
	body := entityDraftBody{Kind: string(draft.Kind), VNum: draft.VNum, Dirty: draft.Diff()}
	value := draft.Effective()
	switch draft.Kind {
	case olc.KindMob:
		body.Mob = &value.Mob
	case olc.KindObject:
		body.Object = &value.Object
	case olc.KindShop:
		body.Shop = &value.Shop
	case olc.KindZone:
		zone := value.Zone.Zone
		zoneBody := &entityZoneDraftBody{Number: zone.Number, Name: zone.Name, TopRoom: zone.TopRoom, Lifespan: zone.Lifespan, ResetMode: zone.ResetMode}
		zoneBody.Commands = make([]entityZoneCommandView, len(zone.Commands))
		for i, command := range zone.Commands {
			zoneBody.Commands[i] = entityZoneCommandView{Position: i, Command: command.Command, IfFlag: command.IfFlag, Arg1: command.Arg1, Arg2: command.Arg2, Arg3: command.Arg3}
		}
		body.Zone = zoneBody
	}
	if !entry.ExpiresAt.IsZero() {
		body.LeaseExpiresAt = entry.ExpiresAt
		remaining := time.Until(entry.ExpiresAt)
		if remaining > 0 {
			body.LeaseRemainingSeconds = int64(remaining / time.Second)
			if remaining%time.Second != 0 {
				body.LeaseRemainingSeconds++
			}
		}
	}
	return body
}

func shopToProto(shop game.Shop) parser.ShopProto {
	return parser.ShopProto{VNum: shop.VNum, Products: append([]int(nil), shop.SellTypes...), BuyProfit: shop.ProfitBuy, SellProfit: shop.ProfitSell, BuyTypes: append([]int(nil), shop.BuyTypes...), BuyWords: append([]string(nil), shop.BuyWords...), Messages: shop.Messages, Temper: shop.Temper, Bitvector: shop.Flags, KeeperVNum: shop.KeeperVNum, WithWho: shop.WithWho, Rooms: append([]int(nil), shop.Rooms...), OpenHour1: shop.OpenHour1, CloseHour1: shop.CloseHour1, OpenHour2: shop.OpenHour2, CloseHour2: shop.CloseHour2}
}

func auditAdapter(logger *audit.AuditLogger) func(olc.AuditEvent) {
	return func(event olc.AuditEvent) {
		if logger != nil {
			logger.Log(audit.AuditEvent{EventType: "olc", User: event.User, IPAddress: event.IPAddress, Action: event.Action, Details: event.Details, Success: event.Success})
		}
	}
}

func entityClaimConflict(writes OLCWriteStateProvider, kind olc.Kind, vnum int, holder olc.Owner) error {
	entry := claimEntry(writes, kind, vnum)
	if entry.OwnerDisplayName == "" && holder != nil {
		entry.OwnerDisplayName = holder.DisplayName()
		entry.OwnerFrontend = holder.Frontend()
	}
	idle := time.Since(entry.ClaimedAt)
	if idle < 0 || entry.ClaimedAt.IsZero() {
		idle = 0
	}
	return &olcConflictError{Message: "OLC resource is already claimed", Holder: entry.OwnerDisplayName, Frontend: string(entry.OwnerFrontend), Idle: idle.Round(time.Second).String(), IdleSecs: int64(idle / time.Second)}
}

func entityLeaseExpired(writes OLCWriteStateProvider, kind olc.Kind, vnum int) error {
	entry := claimEntry(writes, kind, vnum)
	if entry.OwnerDisplayName != "" {
		return entityClaimConflict(writes, kind, vnum, nil)
	}
	return &olcConflictError{Message: "draft lease expired; reopen the draft"}
}
