package steam

import "strconv"

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

func (sid *SteamID) Parse(accid uint32, instance uint32, accountType uint32, universe uint8) {
	*sid = SteamID(uint64(accid) | (uint64(instance&0xFFFFF) << 32) | (uint64(accountType) << 52) | (uint64(universe) << 56))
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
