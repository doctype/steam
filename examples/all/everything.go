package main

import (
	"log"
	"os"
	"time"

	"github.com/asamy45/steam"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	timeTip, err := steam.GetTimeTip()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Time tip: %#v\n", timeTip)

	timeDiff := time.Duration(timeTip.Time - time.Now().Unix())
	session := steam.Session{}
	if err := session.Login(os.Getenv("steamAccount"), os.Getenv("steamPassword"), os.Getenv("steamSharedSecret"), timeDiff); err != nil {
		log.Fatal(err)
	}
	log.Print("Login successful")

	myToken, err := session.GetMyTradeToken()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Trade offer token: %s\n", myToken)

	key, err := session.GetWebAPIKey()
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Key: ", key)

	sid := steam.SteamID(76561198078821986)
	inven, err := session.GetInventory(sid, 730, 2, false)
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range inven {
		log.Printf("Item: %s = %d\n", item.MarketHashName, item.AssetID)
	}

	marketPrices, err := session.GetMarketItemPriceHistory(730, "P90 | Asiimov (Factory New)")
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range marketPrices {
		log.Printf("%s -> %.2f (%s of same price)\n", v.Date, v.Price, v.Count)
	}

	overview, err := session.GetMarketItemPriceOverview(730, "P90 | Asiimov (Factory New)")
	if err != nil {
		log.Fatal(err)
	}

	if overview.Success {
		log.Println("Price overfiew for P90 Asiimov FN:")
		log.Printf("Volume: %s\n", overview.Volume)
		log.Printf("Lowest price: %s Median Price: %s", overview.LowestPrice, overview.MedianPrice)
	}

	sent, _, err := session.GetTradeOffers(steam.TradeFilterSentOffers|steam.TradeFilterRecvOffers, time.Now())
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

	items, err := session.GetTradeReceivedItems(receiptID)
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range items {
		log.Printf("New asset id: %d", item.AssetID)
	}

	identity := os.Getenv("steamIdentitySecret")
	confirmations, err := session.GetConfirmations(identity, time.Now().Add(timeDiff).Unix())
	if err != nil {
		log.Fatal(err)
	}

	for i := range confirmations {
		c := confirmations[i]
		log.Printf("Confirmation ID: %d, Key: %d\n", c.ID, c.Key)
		log.Printf("-> Title %s\n", c.Title)
		log.Printf("-> Receiving %s\n", c.Receiving)
		log.Printf("-> Since %s\n", c.Since)

		tid, err := session.GetConfirmationOfferID(key, c.ID, time.Now().Add(timeDiff).Unix())
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("-> OfferID %d\n", tid)

		err = session.AnswerConfirmation(c, key, "allow", time.Now().Add(timeDiff).Unix())
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Accepted %d\n", c.ID)
	}

	log.Println("Bye!")
}
