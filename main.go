/*!
 * Note!  This is a "test case", it's used for ease of development
 * This will turn into a library.  */
package main

import (
	"log"
	"os"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	twoFactorCode, err := GenerateTwoFactorCode(os.Getenv("steamSharedSecret"))
	if err != nil {
		log.Fatal(err)
	}

	community := Community{}
	if err := community.login(os.Getenv("steamAccount"), os.Getenv("steamPassword"), twoFactorCode); err != nil {
		log.Fatal(err)
	}

	log.Print("Login successful")
	key, err := community.getWebAPIKey()
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Key: ", key)

	offer, err := community.GetTradeOffer(1056256888)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Offer id: %s", offer.ID)
}
