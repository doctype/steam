package main

import "log"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	twoFactorCode, err := GenerateTwoFactorCode("shared secret here")
	if err != nil {
		log.Fatal(err)
	}

	community := Community{}
	if err := community.login("accountname", "password", twoFactorCode); err != nil {
		log.Fatal(err)
	}

	log.Print("Login successful")
	key, err := community.getWebAPIKey()
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Key: ", key)
}
