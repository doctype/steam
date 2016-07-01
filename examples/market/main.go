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

	if resp, err := session.PlaceBuyOrder(730, 0.03, 1, steam.CurrencyUSD, "Chroma 2 Case Key"); err != nil {
		log.Fatal(err)
	} else if resp.ErrCode != 1 {
		log.Printf("unsuccessful buy order placement: %s\n", resp.ErrMsg)
	} else {
		log.Printf("Placed buy order id: %d cancelling...\n", resp.OrderID)
		if err = session.CancelBuyOrder(resp.OrderID); err != nil {
			log.Fatal(err)
		}

		log.Printf("Successfully cancelled buy order %d\n", resp.OrderID)
	}
}
