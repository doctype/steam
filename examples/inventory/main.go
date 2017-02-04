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

	sid := steam.SteamID(76561198078821986)
	apps, err := session.GetInventoryAppStats(sid)
	if err != nil {
		log.Fatal(err)
	}

	for _, v := range apps {
		log.Printf("-- AppID total asset count: %d\n", v.AssetCount)
		for _, context := range v.Contexts {
			log.Printf("-- Items on %d %d (count %d)\n", v.AppID, context.ID, context.AssetCount)
			inven, err := session.GetInventory(sid, v.AppID, context.ID, steam.LangEng)
			if err != nil {
				log.Fatal(err)
			}

			for _, item := range inven {
				log.Printf("Item: %s = %d\n", item.Name.MarketHash, item.AssetID)
			}

			// Wait a bit so we don't get an error.
			time.Sleep(time.Millisecond * 100)
		}
	}

	log.Println("Bye!")
}
