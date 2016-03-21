/*
   Steam Library For Go
   Copyright (C) 2016 Ahmed Samy <f.fallen45@gmail.com>

   This library is free software; you can redistribute it and/or
   modify it under the terms of the GNU Lesser General Public
   License as published by the Free Software Foundation; either
   version 2.1 of the License, or (at your option) any later version.

   This library is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
   Lesser General Public License for more details.

   You should have received a copy of the GNU Lesser General Public
   License along with this library; if not, write to the Free Software
   Foundation, Inc., 51 Franklin St, Fifth Floor, Boston, MA  02110-1301  USA
*/
package main

import "strconv"

type SteamID struct {
	Bits uint64
}

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
	sid.Bits = uint64(accid)
	sid.Bits |= uint64(instance&0xFFFFF) << 32
	sid.Bits |= uint64(accountType) << 52
	sid.Bits |= uint64(universe) << 56
}

func (sid *SteamID) GetAccountID() uint32 {
	return uint32(sid.Bits & 0xFFFFFFFF)
}

func (sid *SteamID) GetAccountInstance() uint32 {
	return uint32((sid.Bits >> 32) & 0xFFFFF)
}

func (sid *SteamID) GetAccountType() uint32 {
	return uint32((sid.Bits >> 52) & 0xF)
}

func (sid *SteamID) GetAccountUniverse() uint32 {
	return uint32((sid.Bits >> 56) & 0xFF)
}

func (sid *SteamID) ToString() string {
	return strconv.FormatUint(sid.Bits, 10)
}
