package main

import (
	"log"
	"os"
	"time"

	"github.com/asamy45/steam"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	session := steam.Session{}
	if err := session.Login(os.Getenv("steamAccount"), os.Getenv("steamPassword"), os.Getenv("steamSharedSecret")); err != nil {
		log.Fatal(err)
	}
	log.Print("Login successful")

	key, err := session.GetWebAPIKey()
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Key: ", key)

	timeTip, err := steam.GetTimeTip()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Time tip: %#v\n", timeTip)
	log.Printf("Their time: %d (ours: %d), offset: %d\n", timeTip.Time, time.Now().Unix(), timeTip.Time-time.Now().Unix())

	identitySecret := os.Getenv("steamIdentitySecret")
	confirmations, err := session.GetConfirmations(identitySecret, time.Now().Unix())
	if err != nil {
		log.Fatal(err)
	}

	for i := range confirmations {
		c := confirmations[i]
		log.Printf("Confirmation ID: %d, Key: %d\n", c.ID, c.Key)
		log.Printf("-> Title %s\n", c.Title)
		log.Printf("-> Receiving %s\n", c.Receiving)
		log.Printf("-> Since %s\n", c.Since)

		tid, err := session.GetConfirmationOfferID(identitySecret, c.ID, time.Now().Unix())
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("-> OfferID %d\n", tid)

		err = session.AnswerConfirmation(c, identitySecret, "allow", time.Now().Unix())
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Accepted %d\n", c.ID)
	}

	log.Println("Bye!")
}
