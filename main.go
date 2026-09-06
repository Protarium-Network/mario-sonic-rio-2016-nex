package main

import (
	"sync"

	"github.com/Protarium-Network/mario-sonic-rio-2016-nex/admin"
	"github.com/Protarium-Network/mario-sonic-rio-2016-nex/nex"
)

func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		nex.StartAuthenticationServer()
	}()
	go func() {
		defer wg.Done()
		nex.StartSecureServer()
	}()

	// Optional leaderboard admin panel; returns immediately if
	// PN_RIO2016_ADMIN_PASSWORD is unset.
	go admin.Start()

	wg.Wait()
}
