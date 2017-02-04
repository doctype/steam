package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/asamy/steam"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	timeTip, err := steam.GetTimeTip()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Time tip: %#v\n", timeTip)

	timeDiff := time.Duration(timeTip.Time - time.Now().Unix())
	session := steam.NewSession(&http.Client{}, "")
	if err := session.Login(os.Getenv("steamAccount"), os.Getenv("steamPassword"), os.Getenv("steamSharedSecret"), timeDiff); err != nil {
		log.Fatal(err)
	}
	log.Print("Login successful")

	profileURL, err := session.GetProfileURL()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Profile URL: %s\n", profileURL)

	profileSetting := uint8( /*Profile*/ steam.PrivacyStatePublic | /*Inventory*/ steam.PrivacyStatePublic<<2 | /*Gifts*/ steam.PrivacyStatePublic<<4)
	if err = session.SetProfilePrivacy(profileURL, steam.CommentSettingSelf, profileSetting); err != nil {
		log.Fatal(err)
	}
	log.Printf("Done editing profile: %d", profileSetting)

	profileInfo := map[string][]string{
		"personaName": {"MasterOfTests"},
		"summary":     {"i am just a test, go away"},
		"customURL":   {"therealtesterOFDOOM"},
	}
	if err = session.SetProfileInfo(profileURL, &profileInfo); err != nil {
		log.Fatal(err)
	}
	log.Print("Done editing profile info")

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

	summaries, err := session.GetPlayerSummaries("76561198078821986")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Profile summaries: %#v\n", summaries[0])

	sid := steam.SteamID(76561198078821986)
	inven, err := session.GetInventory(sid, 730, 2, steam.LangEng)
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range inven {
		log.Printf("Item: %s = %d\n", item.Name.MarketHash, item.AssetID)
	}

	marketPrices, err := session.GetMarketItemPriceHistory(730, "P90 | Asiimov (Factory New)")
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range marketPrices {
		log.Printf("%s -> %.2f (%s of same price)\n", v.Date, v.Price, v.Count)
	}

	overview, err := session.GetMarketItemPriceOverview(730, "DE", "3", "P90 | Asiimov (Factory New)")
	if err != nil {
		log.Fatal(err)
	}

	if overview.Success {
		log.Println("Price overfiew for P90 Asiimov FN:")
		log.Printf("Volume: %s\n", overview.Volume)
		log.Printf("Lowest price: %s Median Price: %s", overview.LowestPrice, overview.MedianPrice)
	}

	resp, err := session.GetTradeOffers(steam.TradeFilterSentOffers, time.Now())
	if err != nil {
		log.Fatal(err)
	}

	var receiptID uint64
	for _, offer := range resp.SentOffers {
		var sid steam.SteamID
		sid.Parse(offer.Partner, steam.AccountInstanceDesktop, steam.AccountTypeIndividual, steam.UniversePublic)

		if receiptID == 0 && len(offer.RecvItems) != 0 && offer.State == steam.TradeStateAccepted {
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
