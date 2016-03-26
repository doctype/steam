package main

import (
	"log"
	"os"

	"github.com/asamy45/steam"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	session := steam.Session{}
	if err := session.Login(os.Getenv("steamAccount"), os.Getenv("steamPassword"), os.Getenv("steamSharedSecret")); err != nil {
		log.Fatal(err)
	}
	log.Print("Login successful")

	sid := steam.SteamID(76561198078821986)
	apps, err := session.GetInventoryAppStats(sid)
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range apps {
		log.Printf("-- AppID total asset count: %d\n", v.AssetCount)
		for _, context := range v.Contexts {
			log.Printf("-- Items on %d %d (count %d)\n", v.AppID, context.ID, context.AssetCount)
			inven, err := session.GetInventory(sid, v.AppID, context.ID, false)
			if err != nil {
				log.Fatal(err)
			}

			for _, item := range inven {
				log.Printf("Item: %s = %d\n", item.MarketHashName, item.AssetID)
			}
		}
	}

	log.Println("Bye!")
}
