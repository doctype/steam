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

	err = community.RevokeWebAPIKey()
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Revoked API Key")

	err = community.RegisterWebAPIKey("test.org")
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Registered new API Key")

	key, err := community.GetWebAPIKey()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Key: %s\n", key)
}
