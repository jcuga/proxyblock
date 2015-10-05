package settings

// Proxy settings are changed via local webserver pages

import (
	"fmt"
	"net/http"
)

func ProxySettingsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `<h1>ProxyBlock Settings</h1><h3>Todo</h3>`)
}
