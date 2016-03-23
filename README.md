# Steam

Steam is a library for interactions with [steam](https://steamcommunity.com), it's written in Go.

## Installation

Installation is simple, and there is only one dependency:

```
go get github.com/PuerkitoBio/goquery
go get github.com/asamy45/steam
```

## Inline Example

```go
package main

import (
	"log"
	"os"

	"github.com/asamy45/gosteam"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	twoFactorCode, err := steam.GenerateTwoFactorCode(os.Getenv("steamSharedSecret"))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Generated two factor code: %s\n", twoFactorCode)

	community := steam.Community{}
	if err := community.Login(os.Getenv("steamAccount"), os.Getenv("steamPassword"), twoFactorCode); err != nil {
		log.Fatal(err)
	}
	log.Print("Login successful")
}
```

## Authors / Thanks to

- [Ahmed Samy](https://github.com/asamy45) <f.fallen45@gmail.com>
- [Mark Samman](https://github.com/marksamman) <mark.samman@gmail.com>

## License

LGPL 2.1
