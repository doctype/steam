package main

import (
	"log"
	"os"
	"time"

	"github.com/asamy45/steam"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	twoFactorCode, err := steam.GenerateTwoFactorCode(os.Getenv("steamSharedSecret"))
	if err != nil {
		log.Fatal(err)
	}

	community := steam.Community{}
	if err := community.Login(os.Getenv("steamAccount"), os.Getenv("steamPassword"), twoFactorCode); err != nil {
		log.Fatal(err)
	}
	log.Print("Login successful")

	key, err := community.GetWebAPIKey()
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Key: ", key)

	sid := steam.SteamID(76561198078821986)
	inven, err := community.GetInventory(&sid, 730, 2, false)
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range inven {
		log.Printf("Item: %s = %d\n", item.MarketHashName, item.AssetID)
	}

	marketPrices, err := community.GetMarketItemPriceHistory(730, "P90 | Asiimov (Factory New)")
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range marketPrices {
		log.Printf("%s -> %.2f (%s of same price)\n", v.Date, v.Price, v.Count)
	}

	overview, err := community.GetMarketItemPriceOverview(730, "P90 | Asiimov (Factory New)")
	if err != nil {
		log.Fatal(err)
	}

	if overview.Success {
		log.Println("Price overfiew for P90 Asiimov FN:")
		log.Printf("Volume: %d\n", overview.Volume)
		log.Printf("Lowest price: %s Median Price: %s", overview.LowestPrice, overview.MedianPrice)
	}

	sent, _, err := community.GetTradeOffers(steam.TradeFilterSentOffers|steam.TradeFilterRecvOffers, time.Now())
	if err != nil {
		log.Fatal(err)
	}

	var receiptID uint64
	for k := range sent {
		offer := sent[k]
		var sid steam.SteamID
		sid.Parse(offer.Partner, steam.AccountInstanceDesktop, steam.AccountTypeIndividual, steam.UniversePublic)

		if receiptID == 0 && len(offer.ReceiveItems) != 0 && offer.State == steam.TradeStateAccepted {
			receiptID = offer.ReceiptID
		}

		log.Printf("Offer id: %d, Receipt ID: %d", offer.ID, offer.ReceiptID)
		log.Printf("Offer partner SteamID 64: %d", uint64(sid))
	}

	items, err := community.GetTradeReceivedItems(receiptID)
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range items {
		log.Printf("New asset id: %d", item.AssetID)
	}

	key, err = steam.GenerateConfirmationCode(os.Getenv("steamIdentitySecret"), "conf")
	if err != nil {
		log.Fatal(err)
	}

	confirmations, err := community.GetConfirmations(key)
	if err != nil {
		log.Fatal(err)
	}

	for i := range confirmations {
		c := confirmations[i]
		log.Printf("Confirmation ID: %d, Key: %d\n", c.ID, c.Key)
		log.Printf("-> Title %s\n", c.Title)
		log.Printf("-> Receiving %s\n", c.Receiving)
		log.Printf("-> Since %s\n", c.Since)

		key, err = steam.GenerateConfirmationCode(os.Getenv("steamIdentitySecret"), "details")
		if err != nil {
			log.Fatal(err)
		}

		tid, err := community.GetConfirmationOfferID(key, c.ID)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("-> OfferID %d\n", tid)

		key, err = steam.GenerateConfirmationCode(os.Getenv("steamIdentitySecret"), "allow")
		err = community.AnswerConfirmation(c, key, "allow")
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Accepted %d\n", c.ID)
	}

	log.Println("Bye!")
}
