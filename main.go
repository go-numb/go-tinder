package main

import (
	"github.com/apex/log"
	"github.com/go-numb/go-tinder/api"
)

const (
	TINDERTOKEN = ""
)

func main() {
	t := new(api.Tinders)
	if err := t.Get(TINDERTOKEN); err != nil {
		log.Error(err)
		return
	}
}
