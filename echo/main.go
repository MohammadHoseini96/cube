package echo

import (
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type Message struct {
	Msg string
}

func main() {
	r := chi.NewRouter()
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		d := json.NewDecoder(r.Body)
		m := Message{}
		if err := d.Decode(&m); err != nil {
			json.NewEncoder(w).Encode(errors.New("unable to decode request body"))
			return
		}
		log.Printf("Received message: %v\n", m.Msg)

		json.NewEncoder(w).Encode(m)
	})
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Health check called")
		w.Write([]byte("OK"))
	})
	r.Get("/healthfailed", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Health check failed")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	})

	srv := &http.Server{
		Addr:    "0.0.0.0:7777",
		Handler: r,
	}

	go func() {
		log.Printf("Listening on %s\n", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	// Setup handler for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	<-c

	log.Println("Shutting down...")
}
