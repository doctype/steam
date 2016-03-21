/*!
 * Note!  This is a "test case", it's used for ease of development
 * This will turn into a library.  */
package main

import (
	"log"
	"os"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	twoFactorCode, err := GenerateTwoFactorCode(os.Getenv("steamSharedSecret"))
	if err != nil {
		log.Fatal(err)
	}

	community := Community{}
	if err := community.login(os.Getenv("steamAccount"), os.Getenv("steamPassword"), twoFactorCode); err != nil {
		log.Fatal(err)
	}

	log.Print("Login successful")
	key, err := community.getWebAPIKey()
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Key: ", key)

	offer := &TradeOffer{
		ReceiveItems: []*EconItem{},
		SendItems: []*EconItem{
			&EconItem{
				AssetID:   5284128177,
				AppID:     730,
				ContextID: 2,
				Amount:    1,
			},
		},
		Message: "hi",
	}
	err = community.SendTradeOffer(offer, SteamID{76561198210171475}, "edjHUXHK")
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Successfully sent trade offer with ID:", offer.ID)
	/*
		sent, _, err := community.GetTradeOffers(TradeFilterSentOffers|TradeFilterRecvOffers, time.Now())
		if err != nil {
			log.Fatal(err)
		}
			heh := false
			for k := range sent {
				offer := sent[k]
				var sid SteamID
				sid.Parse(offer.Partner, AccountInstanceDesktop, AccountTypeIndividual, UniversePublic)

				log.Printf("Offer id: %s", offer.ID)
				log.Printf("Offer partner SteamID 64: %d", sid.Bits)

				if !heh {
					err := community.CancelTradeOffer(strconv.ParseInt(offer.ID, 10, 64))
					if err != nil {
						log.Fatal(err)
					} else {
						heh = true
					}
				}
			}
	*/
}
