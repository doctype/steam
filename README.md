# Steam [![Build Status](https://travis-ci.org/asamy/steam.svg?branch=master)](https://travis-ci.org/asamy/steam)

Steam is a library for interactions with [steam](https://steamcommunity.com), it's written in Go.  
Steam tries to keep-it-simple and does not add extra non-sense.  There are absolutely no internal-polling or such,
      everything is up to you, all it does is wrap around Steam API.

## Why?

- You don't want a library to be "re-trying" automatically
- You don't want a library to be doing your homework
- You are an on-point person and just want stuff that works as-needed.

## Installation

Make sure you have _at least_ Go 1.6 with a GOPATH set then run:

```
go get github.com/PuerkitoBio/goquery
go get github.com/asamy/steam
```

## Example

```go
package main

import (
	"log"
	"os"

	"github.com/asamy/steam"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	timeTip, err := steam.GetTimeTip()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Time tip: %#v\n", timeTip)
	timeDiff := time.Duration(timeTip.Time - time.Now().Unix())
	
	session := steam.Session{}
	if err := session.Login(os.Getenv("steamAccount"), os.Getenv("steamPassword"), os.Getenv("steamSharedSecret"), timeDiff); err != nil {
		log.Fatal(err)
	}
	log.Print("Login successful")
}
```

Find more examples in the examples/ directory.  Even better is to read through the source code, it's simple and
straight-forward to understand.

## Authors

- [Ahmed Samy](https://github.com/asamy) <f.fallen45@gmail.com>
- [Mark Samman](https://github.com/marksamman) <mark.samman@gmail.com>
- [Artemiy Ryabinkov](https://github.com/Furdarius) <getlag@ya.ru>

## License

LGPL 2.1
