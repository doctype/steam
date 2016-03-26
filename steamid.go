package steam

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

type SteamID uint64

const (
	UniverseInvalid = iota
	UniversePublic
	UniverseBeta
	UniverseInternal
	UniverseDev
)

const (
	AccountTypeInvalid = iota
	AccountTypeIndividual
	AccountTypeMultiSeat
	AccountTypeGameServer
	AccountTypeAnonymousGameServer
	AccountTypePending
	AccountTypeContentServer
	AccountTypeClan
	AccountTypeChat
	AccountTypeP2PSuperSeeder
	AccountTypeAnonymous
)

const (
	AccountInstanceAll = iota
	AccountInstanceDesktop
	AccountInstanceConsole
	AccountInstanceWeb
)

const (
	ChatInstanceFlagClan     = 0x80000
	ChatInstanceFlagLobby    = 0x40000
	ChatInstanceFlagMMSLobby = 0x20000
)

var (
	// See Steam Documentation for more information on how this is formatted.
	//		STEAM_X:Y:Z
	//	X + 1 = universe
	//	Y = lowest bit of Account ID
	//	Z = upper 31 bits of Account ID
	legacyRegexp = regexp.MustCompile("STEAM_([0-5]):([0-1]):([0-9]+)")

	// Modern Steam ID
	//		[C:U:A] or [C:U:A:I]
	//	C = account type or instance ID
	//	U = universe
	//	A = account ID
	//	I = instance id if not present, the default instance id for that C value is used.
	modernRegexp = regexp.MustCompile("\\[([a-zA-Z]):([0-5]):([0-9]+)(:[0-9]+)?\\]")

	ErrInvalidSteam2ID = errors.New("invalid input specified for a Steam 2 ID")
	ErrInvalidSteam3ID = errors.New("invalid input specified for a Steam 3 ID")
)

func (sid *SteamID) Parse(accid uint32, instance uint32, accountType uint32, universe uint8) {
	*sid = SteamID(uint64(accid) | (uint64(instance&0xFFFFF) << 32) | (uint64(accountType) << 52) | (uint64(universe) << 56))
}

func (sid *SteamID) ParseSteam2ID(input string) error {
	m := legacyRegexp.FindStringSubmatch(input)
	if m == nil || len(m) < 4 {
		return ErrInvalidSteam2ID
	}

	universe, _ := strconv.ParseUint(string(m[1]), 10, 8)
	lobit, _ := strconv.ParseUint(string(m[2]), 10, 8)
	hibits, _ := strconv.ParseUint(string(m[3]), 10, 32)

	sid.Parse(uint32(lobit|hibits<<1), AccountInstanceDesktop, AccountTypeIndividual, uint8(universe+1))
	return nil
}

func (sid *SteamID) ParseSteam3ID(input string) error {
	m := modernRegexp.FindStringSubmatch(input)
	if m == nil || len(m) < 4 {
		return ErrInvalidSteam3ID
	}

	accountID, _ := strconv.ParseUint(string(m[3]), 10, 32)
	universe, _ := strconv.ParseUint(string(m[2]), 10, 8)

	instance := uint64(AccountInstanceDesktop)
	if len(m) > 5 {
		instance, _ = strconv.ParseUint(string(m[5]), 10, 32)
	}

	accountType := uint32(AccountTypeIndividual)
	switch m[1][0] {
	case 'c':
		instance |= ChatInstanceFlagClan
		accountType = AccountTypeChat
	case 'L':
		instance |= ChatInstanceFlagLobby
		fallthrough
	case 'T':
		accountType = AccountTypeChat
	case 'I':
		accountType = AccountTypeInvalid
	case 'M':
		accountType = AccountTypeMultiSeat
	case 'G':
		accountType = AccountTypeGameServer
	case 'A':
		accountType = AccountTypeAnonymousGameServer
	case 'P':
		accountType = AccountTypePending
	case 'C':
		accountType = AccountTypeContentServer
	case 'g':
		accountType = AccountTypeClan
	case 'a':
		accountType = AccountTypeAnonymous
	}

	sid.Parse(uint32(accountID), uint32(instance), accountType, uint8(universe))
	return nil
}

func (sid *SteamID) GetAccountID() uint32 {
	return uint32(*sid)
}

func (sid *SteamID) GetAccountInstance() uint32 {
	return uint32((*sid >> 32) & 0xFFFFF)
}

func (sid *SteamID) GetAccountType() uint32 {
	return uint32((*sid >> 52) & 0xF)
}

func (sid *SteamID) GetAccountUniverse() uint32 {
	return uint32((*sid >> 56) & 0xFF)
}

func (sid *SteamID) ToString() string {
	return strconv.FormatUint(uint64(*sid), 10)
}

func (sid *SteamID) ToSteam2ID() string {
	universe := sid.GetAccountUniverse() - 1
	accountID := sid.GetAccountID()
	return fmt.Sprintf("STEAM_%d:%d:%d", universe, accountID&1, accountID>>1)
}
