package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/elazarl/goproxy"
	"github.com/elazarl/goproxy/ext/html"

	"github.com/jcuga/golongpoll"

	"github.com/jcuga/proxyblock/proxy/controls"
	"github.com/jcuga/proxyblock/proxy/pagecontrols"
	"github.com/jcuga/proxyblock/proxy/vars"
	"github.com/jcuga/proxyblock/utils"
)

func CreateProxy(whiteList, blackList []*regexp.Regexp, verbose bool,
	whiteListUpdates, blackListUpdates chan string) (*goproxy.ProxyHttpServer, error) {
	// Start longpoll subscription manager
	longpollInterface, lpErr := golongpoll.CreateCustomManager(100, 1000, true)
	if lpErr != nil {
		log.Fatalf("Error creating longpoll manager: %v", lpErr)
	}
	// Create and start control server for controlling proxy behavior
	ctlServer := controls.NewControlServer(vars.ProxyControlPort, longpollInterface.SubscriptionHandler, whiteListUpdates, blackListUpdates)
	ctlServer.Serve()

	// Manually allowed/blocked sites:
	manualWhiteList := make(map[string]bool)
	manualBlackList := make(map[string]bool)

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
		checkWhiteBlackListUpdates(manualWhiteList, manualBlackList,
			whiteListUpdates, blackListUpdates)

		// Now apply whitelist/blacklist rules:
		for _, w := range whiteList {
			if w.MatchString(urlString) {
				// whitelisted by rules, but was this specific URL blacklisted
				// by user?
				if _, ok := manualBlackList[strings.TrimSpace(urlString)]; ok {
					// stop trying to find whitelist matches
					log.Printf("user-DENIED whitelisting:  %s\n", req.URL)
					break
				} else {
					log.Printf("WHITELISTED:  %s\n", req.URL)
					notifyProxyEvent("Allowed", req, longpollInterface)
					return req, nil
				}
			}
		}
		// Check if this was explicitly whitelisted by user:
		if _, ok := manualWhiteList[strings.TrimSpace(urlString)]; ok {
			// no need to consider blacklists, serve content
			log.Printf("user-eplicit WHITELISTED:  %s\n", req.URL)
			notifyProxyEvent("Allowed", req, longpollInterface)
			return req, nil
		}

		// See if we're manually allowing this page thru on time only
		if strings.HasSuffix(urlString, vars.ProxyExceptionString) {
			urlString := urlString[:len(urlString)-len(vars.ProxyExceptionString)]
			u, uErr := url.Parse(urlString)
			if uErr == nil {
				req.URL = u
				log.Printf("MANUALLY ALLOWED: %s\n", req.URL)
				notifyProxyEvent("Manually Allowed", req, longpollInterface)
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
				notifyProxyEvent("Blocked", req, longpollInterface)
				return req, goproxy.NewResponse(req,
					goproxy.ContentTypeHtml, http.StatusForbidden,
					fmt.Sprintf(`<html>
                            <head><title>BLOCKED</title></head>
                            <body>
                                <h1>I pity the fool!</h1>
                                <hr />
                                <h2>Webpage Blocked</h2>
                                <p style="color: black; font-family: monospace; background: #DDDDDD; padding: 20px;">%s</p>
                                <p><a href="%s%s">Continue to Webpage just this once.</a></p>
                                <p>or...</p>
                                <p><a href="http://127.0.0.1:%s/add-wl?url=%s&continue_to_page=yes">Add to Whitelist and continue.</a></p>
                            </body>
                        </html>`, req.URL, req.URL, vars.ProxyExceptionString,
						vars.ProxyControlPort, url.QueryEscape(req.URL.String())))
			}
		}
		log.Printf("NOT MATCHED: (allow by default) %s\n", req.URL)
		notifyProxyEvent("Not matched, default allowed", req, longpollInterface)
		return req, nil
	})

	proxy.OnResponse(goproxy_html.IsHtml).Do(goproxy_html.HandleString(
		func(s string, ctx *goproxy.ProxyCtx) string {
			if strings.HasPrefix(ctx.Req.URL.Host, "http://127.0.0.1:") ||
				strings.HasPrefix(ctx.Req.URL.Host, "http://127.0.0.1/") ||
				strings.HasPrefix(ctx.Req.URL.Host, "127.0.0.1/") ||
				strings.HasPrefix(ctx.Req.URL.Host, "127.0.0.1:") {
				// Don't inject on our own content.
				// TODO: move this logic next to IsHtml so this func
				return s
			}
			// Don't inject iframe into responses that aren't successful
			// ie 2xx response codes.
			// Mainly this is to avoid injecting on our own block page,
			// but it probably doesn't make sense for other failed pages either
			if ctx.Resp.StatusCode < 200 || ctx.Resp.StatusCode >= 300 {
				// show page as-is
				// remember: blocking content is already enforced by this point,
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
					"<div id=\"proxyblock-glass-overlay\" onclick=\"glassClose(this);\" style=\"position: fixed; top: 0; right: 0; left: 0; bottom: 0; background: #000000; opacity: 0.3; z-index: 99999998; display: none;\"></div>" +
					"<div id=\"proxyblock-controls\" style=\"position: fixed; height: 42px; width: 230px; top: 4px; right: 8px; z-index: 99999999;\">" +
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

func notifyProxyEvent(action string, req *http.Request, lpInterface *golongpoll.LongpollInterface) {
	// in the event localhost isn't added to noproxy, don't emit localhost event
	normUrl := strings.ToLower(req.URL.String())
	if strings.HasPrefix(normUrl, "http://127.0.0.1:") ||
		strings.HasPrefix(normUrl, "http://127.0.0.1/") ||
		strings.HasPrefix(normUrl, "127.0.0.1:") ||
		strings.HasPrefix(normUrl, "127.0.0.1/") {
		// no events for you!
		return
	}
	var category string
	if referer := req.Header.Get("Referer"); len(referer) > 0 {
		category = utils.StripProxyExceptionStringFromUrl(referer)
	} else {
		category = utils.StripProxyExceptionStringFromUrl(req.URL.String())
	}
	eventData := action + ": " + req.URL.String()
	if err := lpInterface.Publish(category, eventData); err != nil {
		log.Printf("ERROR: failed to publish event.  error: %q", err)
	}
}

func getParentControlScript() string {
	return `
    <script type="text/javascript">
        function closeControlDetails(wrapper, glass, frame) {
            wrapper.style.height = "42px";
            wrapper.style.width = "230px";
            wrapper.style.maxHeight = null;
            wrapper.style.maxWidth = null;
            glass.style.display = "none";
            frame.setAttribute("scrolling", "no");
        }

        function glassClose(element) {
            var wrapper = document.getElementById("proxyblock-controls");
            var frame = document.getElementById("proxyblock-frame");
            var glass = document.getElementById("proxyblock-glass-overlay");
            closeControlDetails(wrapper, glass, frame);
            // tell child iframe via postMessage to update its dom now that
            // it's supposed to be in closed-details mode:
            var iframewindow = frame.contentWindow ? frame.contentWindow : frame.contentDocument.defaultView;
            iframewindow.postMessage({closeDetails: true}, "*");
        }

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
            var glass = document.getElementById("proxyblock-glass-overlay");
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
                    glass.style.display = "block";
                    frame.setAttribute("scrolling", "auto");
                } else {
                    closeControlDetails(wrapper, glass, frame);
                }
            }
        }, false);
    </script>
    `
}

func checkWhiteBlackListUpdates(whiteListMap, blackListMap map[string]bool,
	whiteListUpdates, blackListUpdates <-chan string) {
	// Right now we're just adding exact url matching... so just regexp escape
	// the new urls and add them to the appropriate white/black list.
	// Try pulling all updates available, break when no more
wlLoop:
	for {
		select {
		case new_url := <-whiteListUpdates:
			fmt.Println("New whitelist entry to add: ", new_url)
			u := strings.TrimSpace(new_url)
			if len(u) > 0 {
				whiteListMap[u] = true
				// also remove this specific url from manual blacklist
				// in case user previously blacklisted it
				delete(blackListMap, u)
			} else {
				log.Printf("ERROR: Invalid whitelist pattern provided: %q",
					new_url)
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
			u := strings.TrimSpace(new_url)
			if len(u) > 0 {
				blackListMap[u] = true
				// also remove this specific url from manual whitelist
				// in case user previously whitelisted it
				delete(whiteListMap, u)
			} else {
				log.Printf("ERROR: Invalid blacklist pattern provided: %q",
					new_url)
			}
		default:
			// No blacklist updates, kill loop
			break blLoop
		}
	}
}
