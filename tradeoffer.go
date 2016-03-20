package main

import "time"

const (
	TradeStateNone = iota
	TradeStateInvalid
	TradeStateActive
	TradeStateAccepted
	TradeStateCountered
	TradeStateExpired
	TradeStateCanceled
	TradeStateDeclined
	TradeStateInvalidItems
	TradeStateCreatedNeedsConfirmation
	TradeStatePendingConfirmation
	TradeStateEmailPending
	TradeStateCanceledByTwoFactor
	TradeStateCanceledConfirmation
	TradeStateEmailCanceled
	TradeStateInEscrow
)

const (
	TradeConfirmationNone = iota
	TradeConfirmationEmail
	TradeConfirmationMobileApp
	TradeConfirmationMobile
)

type EconItem struct {
	assetID    uint64
	instanceID uint64
	classID    uint64
	appID      uint32
	contextID  uint32
	amount     uint32
	marketable bool
	tradable   bool
}

type TradeOffer struct {
	id                 uint64
	receiptID          uint64
	message            string
	state              uint8
	confirmationMethod uint8
	created            time.Time
	updated            time.Time
	expires            time.Time
	escrowEndDate      time.Time
	realTime           bool /* Always false we're not dealing with steam client!  */
}
