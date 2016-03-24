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

	key, err := community.GetWebAPIKey()
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Key: ", key)

	identitySecret := os.Getenv("steamIdentitySecret")
	confirmations, err := community.GetConfirmations(identitySecret)
	if err != nil {
		log.Fatal(err)
	}

	for i := range confirmations {
		c := confirmations[i]
		log.Printf("Confirmation ID: %d, Key: %d\n", c.ID, c.Key)
		log.Printf("-> Title %s\n", c.Title)
		log.Printf("-> Receiving %s\n", c.Receiving)
		log.Printf("-> Since %s\n", c.Since)

		tid, err := community.GetConfirmationOfferID(identitySecret, c.ID)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("-> OfferID %d\n", tid)

		err = community.AnswerConfirmation(c, identitySecret, "allow")
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Accepted %d\n", c.ID)
	}

	log.Println("Bye!")
}
