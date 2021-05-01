package testutils

import (
	"log"

	ma "github.com/multiformats/go-multiaddr"
)

func MustAddProtocol(protocol ma.Protocol) {
	err := ma.AddProtocol(protocol)
	if err != nil {
		log.Fatal(err)
	}
}

func MustCreateProtocol(code int, name string) ma.Protocol {
	return ma.Protocol{
		Code:  code,
		Name:  name,
		VCode: ma.CodeToVarint(code),
	}
}
