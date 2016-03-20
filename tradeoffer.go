package main

const (
	TradeStateNone = iota
	TradeStateInvalid,
	TradeStateActive,
	TradeStateAccepted,
	TradeStateCountered,
	TradeStateExpired,
	TradeStateCanceled,
	TradeStateDeclined,
	TradeStateInvalidItems,
	TradeStateCreatedNeedsConfirmation,
	TradeStatePendingConfirmation,
	TradeStateEmailPending,
	TradeStateCanceledByTwoFactor,
	TradeStateCanceledConfirmation,
	TradeStateEmailCanceled,
	TradeStateInEScrow
)

const (
	TradeConfirmationNone = iota
	TradeConfirmationEmail,
	TradeConfirmationMobileApp,
	TradeConfirmationMobile
)

type EconItem struct {
	assetid    uint64
	instanceid uint64
	classid    uint64
	appid      uint32
	contextid  uint32
	amount     uint32
	marketable bool
	tradable   bool
}

type TradeOffer struct {
	id                 uint64
	receiptId          uint64
	message            string
	state              uint8
	confirmationMethod uint8
	created            time.Time
	updated            time.Time
	expires            time.Time
	escrowEndDate      time.Time
	realTime           bool /* Always false we're not dealing with steam client!  */
}
