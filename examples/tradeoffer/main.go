package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/asamy/steam"
)

func processOffer(session *steam.Session, offer *steam.TradeOffer) {
	var sid steam.SteamID
	sid.ParseDefaults(offer.Partner)

	log.Printf("Offer id: %d, Receipt ID: %d", offer.ID, offer.ReceiptID)
	log.Printf("Offer partner SteamID 64: %d", uint64(sid))
	if offer.State == steam.TradeStateAccepted {
		items, err := session.GetTradeReceivedItems(offer.ReceiptID)
		if err != nil {
			log.Printf("error getting items: %v", err)
		} else {
			for _, item := range items {
				log.Printf("Item: %#v", item)
			}
		}
	}
}

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

	key, err := session.GetWebAPIKey()
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Key: ", key)

	resp, err := session.GetTradeOffers(
		steam.TradeFilterSentOffers|steam.TradeFilterRecvOffers,
		time.Now(),
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, offer := range resp.SentOffers {
		processOffer(session, offer)
	}
	for _, offer := range resp.ReceivedOffers {
		processOffer(session, offer)
	}

	log.Println("Bye!")
}
