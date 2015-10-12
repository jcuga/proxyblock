package proxy

import (
	"github.com/elazarl/goproxy"
	"github.com/elazarl/goproxy/ext/html"

	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jcuga/proxyblock/longpolling"
	"github.com/jcuga/proxyblock/proxy/controls"
	"github.com/jcuga/proxyblock/proxy/pagecontrols"
	"github.com/jcuga/proxyblock/proxy/vars"
	"github.com/jcuga/proxyblock/utils"
)

func CreateProxy(whiteList, blackList []*regexp.Regexp, verbose bool,
	whiteListUpdates, blackListUpdates chan string) (*goproxy.ProxyHttpServer, error) {
	// Start longpoll subscription manager
	eventChan, eventAjaxHandler := longpolling.StartLongpollManager()
	// Create and start control server for controlling proxy behavior
	ctlServer := controls.NewControlServer(vars.ProxyControlPort, eventAjaxHandler, whiteListUpdates, blackListUpdates)
	ctlServer.Serve()

	// Create and start our content blocking proxy:
	proxy := goproxy.NewProxyHttpServer()
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		// Prevent upgrades to https so we can easily see everything as plain
		if req.URL.Scheme == "https" {
			req.URL.Scheme = "http"
		}
		urlString := req.URL.String()

		// Check for any updates to our whitelist/blacklist values
		whiteList, blackList = checkWhiteBlackListUpdates(whiteList, blackList, whiteListUpdates,
			blackListUpdates)
		// Now apply whitelist/blacklist rules:
		for _, w := range whiteList {
			if w.MatchString(urlString) {
				log.Printf("WHITELISTED:  %s\n", req.URL)
				notifyProxyEvent("Allowed", req, eventChan)
				return req, nil
			}
		}
		// See if we're manually allowing this page thru
		if strings.HasSuffix(urlString, vars.ProxyExceptionString) {
			urlString := urlString[:len(urlString)-len(vars.ProxyExceptionString)]
			u, uErr := url.Parse(urlString)
			if uErr == nil {
				req.URL = u
				log.Printf("MANUALLY ALLOWED: %s\n", req.URL)
				notifyProxyEvent("Manually Allowed", req, eventChan)
				return req, nil
			} else {
				log.Printf("ERROR trying to rewrite URL. Url: %s, Error: %s", urlString, uErr)
				return req, goproxy.NewResponse(req,
					goproxy.ContentTypeHtml, http.StatusForbidden,
					fmt.Sprintf(`<html>
                            <head><title>BAD URL</title></head>
                            <body>
                                <h1>Ehhh.... wut?</h1>
                                <hr />
                                <h2>Error rewriting URL:</h2>
                                <p style="color: black; font-family: monospace; background: #DDDDDD; padding: 20px;">%s</p>
                                <p>Error:</p>
                                <p style="color: red; font-family: monospace; background: #DDDDDD; padding: 20px;">%s</p>
                            </body>
                        </html>`, urlString, uErr))
			}
		}
		for _, b := range blackList {
			if b.MatchString(urlString) {
				log.Printf("BLACKLISTED:  %s\n", req.URL)
				notifyProxyEvent("Blocked", req, eventChan)
				return req, goproxy.NewResponse(req,
					goproxy.ContentTypeHtml, http.StatusForbidden,
					fmt.Sprintf(`<html>
                            <head><title>BLOCKED</title></head>
                            <body>
                                <h1>I pitty the fool!</h1>
                                <hr />
                                <h2>Webpage Blocked</h2>
                                <p style="color: black; font-family: monospace; background: #DDDDDD; padding: 20px;">%s</p>
                                <p><a href="%s%s">Continue to Webpage</a></p>
                            </body>
                        </html>`, req.URL, req.URL, vars.ProxyExceptionString))
			}
		}
		log.Printf("NOT MATCHED: (allow by default) %s\n", req.URL)
		notifyProxyEvent("Not matched, default allowed", req, eventChan)
		return req, nil
	})

	proxy.OnResponse(goproxy_html.IsHtml).Do(goproxy_html.HandleString(
		func(s string, ctx *goproxy.ProxyCtx) string {
			if strings.HasPrefix(ctx.Req.URL.Host, "127.0.0.1") || strings.HasPrefix(ctx.Req.URL.Host, "localhost") {
				// Don't inject on our own content.
				// TODO: move this logic next to IsHtml so this func
				// never called?
				return s
			}
			match := vars.StartBodyTagMatcher.FindIndex([]byte(s))
			if match != nil && len(match) >= 2 {
				// TODO: make this more efficient by using a stream or some sort
				// of stringbuilder like thing that doesn't require mashing
				// giant strings together.
				return s[:match[1]] +
					// TODO: should this script get injected after the iframe to prevent a potential race condition?
					getParentControlScript() +
					"<div id=\"proxyblock-controls\" style=\"position: fixed; height: 42px; width: 230px !important; top: 4px; right: 8px; z-index: 99999999;\">" +
					"<iframe id=\"proxyblock-frame\" scrolling=\"no\" style=\"overflow: hidden; background-color: #FFFFFF; border: 2px solid black; width: 100%; height: 100%;\" " +
					"src=\"http://127.0.0.1:" + vars.ProxyControlPort + pagecontrols.GetPageControlsUrl(ctx.Req.URL.String()) +
					"\"></iframe>" +
					"</div>" +
					s[match[1]:]
			} else {
				log.Printf("WARNING: No starting body tag found, must not be html, no injection.")
				return s
			}
		}))
	proxy.Verbose = verbose
	return proxy, nil
}

func notifyProxyEvent(action string, req *http.Request, events chan longpolling.Event) {
	var category string
	// TODO: comments about how longpoll subscriptions for a given referrer (or
	// url when not a referred page).  This way we can show all content allowed/blocked
	// for a given page.
	if referer := req.Header.Get("Referer"); len(referer) > 0 {
		category = utils.StripProxyExceptionStringFromUrl(referer)
	} else {
		category = utils.StripProxyExceptionStringFromUrl(req.URL.String())
	}
	event := longpolling.Event{time.Now(), category, action + ": " + req.URL.String()}
	events <- event
}

func getParentControlScript() string {
	return `
    <script type="text/javascript">
        // Here "addEventListener" is for standards-compliant web browsers and "attachEvent" is for IE Browsers.
        var eventMethod = window.addEventListener ? "addEventListener" : "attachEvent";
        var eventer = window[eventMethod];
        // onmessage for attachEvent, message for addEventListener
        var messageEvent = eventMethod == "attachEvent" ? "onmessage" : "message";
        // Listen to message from child IFrame window
        eventer(messageEvent, function (e) {
            if (e.origin.slice(0, 17) !== "http://127.0.0.1:" && e.origin.slice(0, 18) !== "https://127.0.0.1:") {
                return;
            }
            var wrapper = document.getElementById("proxyblock-controls");
            var frame = document.getElementById("proxyblock-frame");
            if (e.data.upTop !== undefined) {
                // user toggled control position.  reposition:
                if (e.data.upTop) {
                    wrapper.style.bottom = null;
                    wrapper.style.top = "4px";
                } else {
                    wrapper.style.top = null;
                    wrapper.style.bottom = "8px";
                }
            }
            if (e.data.expanded !== undefined) {
                // user toggled control exanded state.
                if (e.data.expanded) {
                    wrapper.style.height = "90%";
                    wrapper.style.width = "90%";
                    wrapper.style.maxHeight = "1000px";
                    wrapper.style.maxWidth = "900px";
                    frame.setAttribute("scrolling", "auto");
                } else {
                    wrapper.style.height = "42px";
                    wrapper.style.width = "230px";
                    wrapper.style.maxHeight = null;
                    wrapper.style.maxWidth = null;
                    frame.setAttribute("scrolling", "no");
                }
            }
        }, false);
    </script>
    `
}

func checkWhiteBlackListUpdates(whiteList, blackList []*regexp.Regexp, whiteListUpdates,
	blackListUpdates <-chan string) (wl, bl []*regexp.Regexp) {
	// Right now we're just adding exact url matching... so just regexp escape
	// the new urls and add them to the appropriate white/black list.
	// Try pulling all updates available, break when no more
wlLoop:
	for {
		select {
		case new_url := <-whiteListUpdates:
			fmt.Println("New whitelist entry to add: ", new_url)
			if r, err := regexp.Compile(regexp.QuoteMeta(new_url)); err == nil {
				// TODO: only send if regexp not in list already!
				whiteList = append(whiteList, r)
			} else {
				// since we're regexp escaping (via QuoteMeta) this should never fail
				log.Fatalf("Invalid whitelist pattern added: %q",
					regexp.QuoteMeta(new_url))
			}
		default:
			// No whitelist updates, kill loop
			break wlLoop
		}
	}
	// Try pulling all updates available, break when no more
blLoop:
	for {
		select {
		case new_url := <-blackListUpdates:
			fmt.Println("New blacklist entry to add: ", new_url)
			if r, err := regexp.Compile(regexp.QuoteMeta(new_url)); err == nil {
				// TODO: only send if regexp not in list already!
				blackList = append(blackList, r)
			} else {
				// since we're regexp escaping (via QuoteMeta) this should never fail
				log.Fatalf("Invalid blacklist pattern added: %q",
					regexp.QuoteMeta(new_url))
			}
		default:
			// No blacklist updates, kill loop
			break blLoop
		}
	}
	return whiteList, blackList
}
