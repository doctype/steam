/*!
 * Note!  This is a "test case", it's used for ease of development
 * This will turn into a library.  */
package main

import (
	"log"
	"os"
	"time"
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

	sent, _, err := community.GetTradeOffers(TradeFilterSentOffers|TradeFilterRecvOffers, time.Now())
	if err != nil {
		log.Fatal(err)
	}

	for k := range sent {
		offer := sent[k]
		var sid SteamID
		sid.Parse(offer.Partner, AccountInstanceDesktop, AccountTypeIndividual, UniversePublic)

		log.Printf("Offer id: %s", offer.ID)
		log.Printf("Offer partner SteamID 64: %d", sid.Bits)
	}
}
