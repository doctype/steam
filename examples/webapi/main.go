package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/doctype/steam"
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

	err = session.RevokeWebAPIKey()
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Revoked API Key")

	key, err := session.RegisterWebAPIKey("test.org")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Registered new API Key: %s", key)

	ownedGames, err := session.GetOwnedGames(steam.SteamID(76561198078821986), false, true)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Games count: %d\n", ownedGames.Count)
	for _, game := range ownedGames.Games {
		log.Printf("Game: %d 2 weeks play time: %d\n", game.AppID, game.Playtime2Weeks)
	}
}
