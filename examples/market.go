package main

import (
	"log"
	"os"

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
}
