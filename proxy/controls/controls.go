package controls

// Proxy behavior is controlled via a local webserver

import (
	"fmt"
	"log"
	"net/http"

	"github.com/jcuga/proxyblock/proxy/pagecontrols"
	"github.com/jcuga/proxyblock/proxy/settings"
)

type HTTPServer struct {
	port  string
	https *http.Server
}

func (s *HTTPServer) Serve() {
	go s.https.ListenAndServe()
}

func NewControlServer(port string, eventAjaxHandler func(w http.ResponseWriter, r *http.Request), whiteListUpdates, blackListUpdates chan<- string) *HTTPServer {
	s := &HTTPServer{port, &http.Server{Addr: "127.0.0.1:" + port, Handler: nil}}
	mux := http.NewServeMux()
	mux.HandleFunc(pagecontrols.ProxyPageControlsUrl, pagecontrols.PageControlsHandler)
	mux.HandleFunc("/events", eventAjaxHandler)
	mux.HandleFunc("/proxy-settings", settings.ProxySettingsHandler)
	mux.HandleFunc("/add-wl", getAddListItemHandler(whiteListUpdates))
	mux.HandleFunc("/add-bl", getAddListItemHandler(blackListUpdates))
	// TODO: remove-wl url
	// TODO: remove-bl url
	s.https.Handler = mux
	return s
}

func getAddListItemHandler(updateList chan<- string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// Don't cache response:
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1.
		w.Header().Set("Pragma", "no-cache")                                   // HTTP 1.0.
		w.Header().Set("Expires", "0")                                         // Proxies.
		new_url := r.URL.Query().Get("url")
		if len(new_url) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "400 Bad request.")
		}
		log.Printf("Adding item to white/black list: %s", new_url)
		// send new url to proxy and it will add it to it's white/black list
		updateList <- new_url
		// if this was a "Add to whitelist and continue" link click from the
		// block page, then we'll want to let the user continue to the
		// original page
		continue_to := r.URL.Query().Get("continue_to_page")
		if continue_to == "yes" {
			http.Redirect(w, r, new_url, 301)
		} else {
			fmt.Fprint(w, "200 ok")
		}
	}
}

// TODO: getRemoveListItemHandler
