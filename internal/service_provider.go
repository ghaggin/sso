package internal

import (
	"bufio"
	"encoding/gob"
	"fmt"
	"net/http"
	"os"
	"time"

	scs "github.com/alexedwards/scs/v2"
	"github.com/crewjam/saml/samlsp"
	"github.com/go-chi/chi/v5"
)

var sessionManager *scs.SessionManager

func newSessionManager() {
	gob.Register(&User{})

	sessionManager = scs.New()
	sessionManager.Lifetime = time.Minute * 3
}

func ServiceProvider(p int, idpURL string) {
	fmt.Printf("running service provider on port %v\n", p)
	port := fmt.Sprintf(":%v", p)

	samlSP, err := newSamlSP(port, idpURL)
	if err != nil {
		panic(err)
	}

	newSessionManager()

	root := chi.NewRouter()
	root.Use(sessionManager.LoadAndSave)

	// Auth
	root.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Get("/", home)
		r.Get("/attr", getAttrVals)
	})

	// No Auth
	root.Group(func(r chi.Router) {
		r.HandleFunc("/login", login)
		r.HandleFunc("/saml/login", func(w http.ResponseWriter, r *http.Request) {
			samlSP.HandleStartAuthFlow(w, r)
		})

		r.Get("/saml/metadata", samlSP.ServeMetadata)
		r.Post("/saml/acs", samlSP.ServeACS)

		r.Handle("/static/*", http.StripPrefix("/static", http.FileServer(http.Dir("web/static/"))))
	})

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

func home(w http.ResponseWriter, r *http.Request) {
	user, ok := sessionManager.Get(r.Context(), "user").(*User)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}

	renderTemplate(w, r, "home.html", &templateData{
		PageTitle: "home",
		UID:       user.UID,
	})
}

func login(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, r, "login.html", &templateData{
		PageTitle: "login",
	})
}

func getAttrVals(w http.ResponseWriter, r *http.Request) {
	s := samlsp.SessionFromContext(r.Context())
	if s == nil {
		fmt.Fprint(w, "s is null")
		return
	}

	sa, ok := s.(samlsp.SessionWithAttributes)
	if !ok {
		fmt.Fprint(w, "couldn't cast sa")
		return
	}
	attrs := sa.GetAttributes()

	for attr, attrVal := range attrs {
		fmtAttrVal := ""
		for i, v := range attrVal {
			if i == 0 {
				fmtAttrVal = v
				continue
			}
			fmtAttrVal = fmtAttrVal + ", " + v
		}
		fmt.Fprintf(w, "%v: %v\n", attr, fmtAttrVal)
	}
}

type User struct {
	UID string
}

// Presence of a user in the context indicates auth
func requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := sessionManager.Get(r.Context(), "user").(*User)
		if !ok {
			http.Redirect(w, r, "/saml/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}
