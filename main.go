package main

import "fmt"

func main() {
	community := Community{}
	err := community.login("accountname", "password", GenerateTwoFactorCode("shared secret here"))
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	fmt.Println("Login successful")
	key, err := community.getWebApiKey()
	if err != nil {
		fmt.Println("Error: ", err)
	}

	fmt.Println("Key: ", key)
}
