package main

import (
	"log"
	"os"

	"github.com/asamy45/steam"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	community := steam.Community{}
	if err := community.Login(os.Getenv("steamAccount"), os.Getenv("steamPassword"), os.Getenv("steamSharedSecret")); err != nil {
		log.Fatal(err)
	}
	log.Print("Login successful")

	sid := steam.SteamID(76561198078821986)
	inven, err := community.GetInventory(&sid, 730, 2, false)
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range inven {
		log.Printf("Item: %s = %d\n", item.MarketHashName, item.AssetID)
	}

	log.Println("Bye!")
}
