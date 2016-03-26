# Steam

Steam is a library for interactions with [steam](https://steamcommunity.com), it's written in Go.  
Steam tries to keep-it-simple and does not add extra non-sense.  There are absolutely no internal-polling or such,
      everything is up to you, all it does is wrap around Steam API.

## Why?

- You don't want a library to be "re-trying" automatically
- You don't want a library to be doing your homework
- You are an on-point person and just want stuff that works as-needed.

## Installation

Installation is simple, and there is only one dependency:

```
go get github.com/PuerkitoBio/goquery
go get github.com/asamy45/steam
```

## Example

```go
package main

import (
	"log"
	"os"

	"github.com/asamy45/steam"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	community := steam.Community{}
	if err := community.Login(os.Getenv("steamAccount"), os.Getenv("steamPassword"), os.Getenv("steamSharedSecret")); err != nil {
		log.Fatal(err)
	}
	log.Print("Login successful")
}
```

Find more examples in the examples/ directory.  Even better is to read through the source code, it's simple and
straight-forward to understand.

## Authors

- [Ahmed Samy](https://github.com/asamy45) <f.fallen45@gmail.com>
- [Mark Samman](https://github.com/marksamman) <mark.samman@gmail.com>

## License

LGPL 2.1
