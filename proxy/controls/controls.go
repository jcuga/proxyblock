package controls

// Proxy behavior is controlled via a local webserver

import (
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

func NewControlServer(port string, eventAjaxHandler func(w http.ResponseWriter, r *http.Request)) *HTTPServer {
    s := &HTTPServer{port, &http.Server{Addr: "127.0.0.1:" + port, Handler: nil}}
    mux := http.NewServeMux()
    mux.HandleFunc(pagecontrols.ProxyPageControlsUrl, pagecontrols.PageControlsHandler)
    mux.HandleFunc("/events", eventAjaxHandler)
    mux.HandleFunc("/proxy-settings", settings.ProxySettingsHandler)
    s.https.Handler = mux
    return s
}
