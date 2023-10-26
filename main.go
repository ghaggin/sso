package main

import (
	"flag"

	"github.com/ghaggin/test-idp/internal"
)

func main() {
	var mode = flag.String("mode", "", "either sp or idp")
	var port = flag.Int("port", 0, "port to run the server on")
	var idpURL = flag.String("idp", "", "idp url")
	flag.Parse()

	if *mode == "sp" {
		internal.ServiceProvider(*port, *idpURL)
	} else if *mode == "idp" {
		internal.IdentityProvider(*port)
	} else {
		panic("unrecognized mode")
	}
}
