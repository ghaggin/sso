package internal

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/crewjam/saml/samlidp"
	"github.com/go-chi/chi/v5"
)

func IdentityProvider(p int) {
	fmt.Printf("running identity provider on port %v\n", p)
	port := fmt.Sprintf(":%v", p)

	baseUrl, err := url.Parse("http://localhost" + port)
	if err != nil {
		panic(err)
	}

	key, cert, err := getKeyPair("idp")
	if err != nil {
		panic(err)
	}

	idpServer := newSamlIdentityProvider(samlidp.Options{
		URL:         *baseUrl,
		Key:         key,
		Certificate: cert,
		Store:       &samlidp.MemoryStore{},
	})

	root := chi.NewRouter()
	root.Get("/metadata", func(w http.ResponseWriter, r *http.Request) {
		idpServer.IDP.ServeMetadata(w, r)
	})
	root.HandleFunc("/sso", func(w http.ResponseWriter, r *http.Request) {
		idpServer.IDP.ServeSSO(w, r)
	})

	root.Get("/service", idpServer.HandleGetService)
	root.Put("/service", idpServer.HandlePutService)
	root.Post("/service", idpServer.HandlePutService)

	go http.ListenAndServe("localhost"+port, root)

	fmt.Println("press q + <Enter> to exit...")
	for {
		b, err := bufio.NewReader(os.Stdin).ReadBytes('\n')
		if err != nil {
			fmt.Println(err)
		}
		if string(b) == "q\n" {
			fmt.Println("exiting")
			return
		}
	}
}
