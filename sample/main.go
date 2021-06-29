package main

import (
	"errors"
	"log"
	"github.com/tinygoprogs/circuitbreaker"
	"time"
)

func init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
}

var i = 0

func NastyThirdPartyCode() error {
	log.Print("NastyThirdPartyCode")
	if i%3 == 0 {
		return errors.New("whoopsie")
	} else {
		time.Sleep(time.Millisecond * 300)
	}
	return nil
}

func FallbackToThirdPartyCode() error {
	log.Print("FallbackToThirdPartyCode")
	time.Sleep(300 * time.Millisecond)
	return nil
}

func main() {
	cb := circuitbreaker.NewCircuitBreaker(&circuitbreaker.Config{
		FallbackFunc: FallbackToThirdPartyCode,
	})
	for {
		err := cb.Execute(NastyThirdPartyCode)
		if err != nil {
			log.Print(err)
		}
	}
}
