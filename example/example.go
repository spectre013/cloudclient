package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/spectre013/cloudclient"
	"log"
	"net/http"
	"sync"
	"time"
)

var property *cloudclient.Property
var srv *http.Server

func main() {
	property = cloudclient.Client("http://localhost:8888", "rss-entry-service", "dev")
	property.Updated = false
	go refresh()
	srv = startHttpServer()

	wg := sync.WaitGroup{} // Use a WaitGroup to block main() exit
	wg.Add(1)
	wg.Wait()
}

func refresh() {
	for {
		time.Sleep(time.Second * 15)
		property = property.GetProperty()

		if property.HasUpdate() {
			property.Updated = false
			log.Println("Properties are updated restarting server to pick up changes")
			if err := srv.Shutdown(context.TODO()); err != nil {
				panic(err) // failure/timeout shutting down the server gracefully
			}
			srv = startHttpServer()
		}
	}
}

func startHttpServer() *http.Server {

	router := http.NewServeMux()
	router.Handle("/", Log(http.HandlerFunc(index)))
	router.Handle("/config", Log(http.HandlerFunc(config)))
	log.Println("Serving search from: ", property.Properties["ui.search.servdir"])
	router.Handle("/search", Log(http.StripPrefix("/search", http.FileServer(http.Dir(property.Properties["ui.search.servdir"])))))

	srv := &http.Server{
		Addr:    ":" + property.Properties["ui.search.port"],
		Handler: router,
	}

	go func() {
		// returns ErrServerClosed on graceful close
		log.Println("Starting server: " + srv.Addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			// NOTE: there is a chance that next line won't have time to run,
			// as main() doesn't wait for this goroutine to stop. don't use
			// code with race conditions like these for production. see post
			// comments below on more discussion on how to handle this.
			log.Fatalf("ListenAndServe(): %s", err)
		}
	}()

	// returning reference so caller can call Shutdown()
	return srv
}

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello")
}
func config(w http.ResponseWriter, r *http.Request) {
	rs, err := json.Marshal(property.Properties)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Fprint(w, string(rs))
}

func Log(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}
