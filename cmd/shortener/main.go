package main

import (
	"github.com/go-chi/chi"
	"net/http"
)

func main() {
	// The HTTP Server
	//config := configrw.NewConfig()
	//myShortener := shortener.NewShortener()
	r := chi.NewRouter()
	r.Route("/", Route())
	server := &http.Server{Addr: "127.0.0.1:8080", Handler: r}

	// Server run context
	/*	serverCtx, serverStopCtx := context.WithCancel(context.Background())
		fmt.Println("addr: " + config.SrvAddr())
		// Listen for syscall signals for process to interrupt/quit
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		go func() {
			<-sig

			// Shutdown signal with grace period of 30 seconds
			shutdownCtx, cancelCtx := context.WithTimeout(serverCtx, 30*time.Second)

			go func() {
				<-shutdownCtx.Done()
				if shutdownCtx.Err() == context.DeadlineExceeded {
					defer cancelCtx()
					fmt.Println("graceful shutdown timed out.. forcing exit.")
				}
			}()

			// Trigger graceful shutdown
			myShortener.FlushStorage()
			err := server.Shutdown(shutdownCtx)
			if err != nil {
				fmt.Println(err.Error())
			}
			serverStopCtx()
		}()*/

	// Run the server
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		panic(err.Error())
	}

	// Wait for server context to be stopped
	//<-serverCtx.Done()
}

func Test(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusGone)
}

func Route() func(r chi.Router) {
	/*return func(r chi.Router) {
		r.Get("/", Test)
	}*/
	return func(r chi.Router) {
		r.Method("GET", "/", http.HandlerFunc(Test))
	}
}
