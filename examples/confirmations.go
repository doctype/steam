package main

import (
	"log"
	"os"
	"time"

	"github.com/asamy45/steam"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	community := steam.Community{}
	if err := community.Login(os.Getenv("steamAccount"), os.Getenv("steamPassword"), os.Getenv("steamSharedSecret")); err != nil {
		log.Fatal(err)
	}
	log.Print("Login successful")

	key, err := community.GetWebAPIKey()
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
	confirmations, err := community.GetConfirmations(identitySecret, time.Now().Unix())
	if err != nil {
		log.Fatal(err)
	}

	for i := range confirmations {
		c := confirmations[i]
		log.Printf("Confirmation ID: %d, Key: %d\n", c.ID, c.Key)
		log.Printf("-> Title %s\n", c.Title)
		log.Printf("-> Receiving %s\n", c.Receiving)
		log.Printf("-> Since %s\n", c.Since)

		tid, err := community.GetConfirmationOfferID(identitySecret, c.ID, time.Now().Unix())
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("-> OfferID %d\n", tid)

		err = community.AnswerConfirmation(c, identitySecret, "allow", time.Now().Unix())
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Accepted %d\n", c.ID)
	}

	log.Println("Bye!")
}
