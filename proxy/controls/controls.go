package controls

// Proxy behavior is controlled via a local webserver

import (
	"fmt"
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

func NewControlServer(port string,
	eventAjaxHandler func(w http.ResponseWriter, r *http.Request),
	whiteListUpdates, blackListUpdates chan<- string) *HTTPServer {
	s := &HTTPServer{port, &http.Server{Addr: "127.0.0.1:" + port, Handler: nil}}
	mux := http.NewServeMux()
	mux.HandleFunc(pagecontrols.ProxyPageControlsUrl, pagecontrols.PageControlsHandler)
	mux.HandleFunc("/events", eventAjaxHandler)
	mux.HandleFunc("/proxy-settings", settings.ProxySettingsHandler)
	mux.HandleFunc("/add-wl", getAddListItemHandler(whiteListUpdates))
	mux.HandleFunc("/add-bl", getAddListItemHandler(blackListUpdates))
	// TODO: remove-wl
	// TODO: remove-bl
	s.https.Handler = mux
	return s
}

func getAddListItemHandler(updateList chan<- string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		new_url := r.URL.Query().Get("url")
		if len(new_url) < 1 {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "400 Bad request.")
		}
		// send new url to proxy and it will add it to it's white/black list
		updateList <- new_url
		fmt.Fprint(w, "200 ok")
	}
}

// TODO: getRemoveListItemHandler
