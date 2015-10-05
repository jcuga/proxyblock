package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/jcuga/proxyblock/proxy"
	"github.com/jcuga/proxyblock/utils"
)

func main() {
	verbose := flag.Bool("v", false, "should every proxy request be logged to stdout")
	addr := flag.String("addr", "127.0.0.1:3128", "proxy listen address")
	whitelistFilename := flag.String("wl", "whitelist.txt", "file of regexes to whitelist request urls (overrides blacklist)")
	blacklistFilename := flag.String("bl", "blacklist.txt", "file of regexes to blacklistlist request urls")

	flag.Parse()
	whiteList, wlErr := utils.GetRegexlist(*whitelistFilename)
	if wlErr != nil {
		log.Fatalf("Could not load whitelist. Error: %s", wlErr)
	}
	blackList, blErr := utils.GetRegexlist(*blacklistFilename)
	if blErr != nil {
		log.Fatalf("Could not load blacklist. Error: %s", blErr)
	}

	proxy, err := proxy.CreateProxy(whiteList, blackList, *verbose)
	if err != nil {
		log.Fatalf("Error creating proxy: %s", err)
	} else {
		// Start proxy (this call is blocking)
		log.Fatal(http.ListenAndServe(*addr, proxy))
	}
}
