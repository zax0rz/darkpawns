package game

// Additional player flag bit constants — from src/structs.h PLR_* defines.
// PlrOutlaw, PlrNODELETE, PlrCRYO, PlrWerewolf, PlrVampire already exist in other_helpers.go.
const (
	PlrOpen      = 1
	PlrFrozen    = 2
	PlrDontset   = 3 // Don't EVER set (ISNPC bit)
	PlrWriting   = 4
	PlrMailing   = 5
	PlrCrash     = 6
	PlrSiteok    = 7
	PlrNoshout   = 8
	PlrNotitle   = 9
	PlrDeleted   = 10
	PlrLoadroom  = 11
	PlrNowizlist = 12
	PlrInvstart  = 14
	PlrIt        = 18
	PlrChosen    = 19
	PlrRemort    = 20
	PlrExtract   = 21
)
