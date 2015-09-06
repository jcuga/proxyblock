package main

import (
    "github.com/elazarl/goproxy"
    "github.com/elazarl/goproxy/ext/html"

    "log"
    "flag"
    "bufio"
    "os"
    "fmt"
    "strings"
    "regexp"
    "net/http"
    "net/url"
    "time"

    "github.com/jcuga/proxyblock/longpolling"
)

var (
    startBodyTagMatcher = regexp.MustCompile(`(?i:<body.*>)`)
    controlPort = "8380"
    proxyExceptionString = "LOL-WHUT-JUST-DOIT-DOOD"
)

type HTTPServer struct {
    port string
    https *http.Server
}

func (s *HTTPServer) Serve() {
    go s.https.ListenAndServe()
}

func NewControlServer(port string, eventAjaxHandler func(w http.ResponseWriter, r *http.Request) ) (*HTTPServer) {
    s := &HTTPServer{port,&http.Server{Addr: "127.0.0.1:" + port, Handler: nil } }
    mux := http.NewServeMux()
    mux.HandleFunc("/page-menu", pageMenuHandler)
    mux.HandleFunc("/events", eventAjaxHandler)
    s.https.Handler = mux
    return s
}

// Since our event subscriptions (longpoll) are based on a 'category' which is
// the URL/referer, when we add a proxy exception string to manually bypass
// content blocking, the string on the end of the URL will cause a mismatch and
// we'll never get our content accepted/blocked notifications
func stripProxyExceptionStringFromUrl(url string) string {
    if (strings.HasSuffix(url, proxyExceptionString)) {
        return url[:len(url) - len(proxyExceptionString)]
    } else {
        return url
    }
}

func notifyEvent(action string, req *http.Request, events chan longpolling.Event) {
    var category string
    // TODO: comments about how longpoll subscriptions for a given referrer (or
    // url when not a referred page).  This way we can show all content allowed/blocked
    // for a given page.
    if referer := req.Header.Get("Referer") ; len(referer) > 0 {
        category = stripProxyExceptionStringFromUrl(referer)
    } else {
        category = stripProxyExceptionStringFromUrl(req.URL.String())
    }
    event := longpolling.Event{time.Now(), category, action + ": " + req.URL.String()}
    events <- event
}

func main() {
    verbose := flag.Bool("v", false, "should every proxy request be logged to stdout")
    addr := flag.String("addr", "127.0.0.1:3128", "proxy listen address")
    whitelistFilename := flag.String("wl", "whitelist.txt", "file of regexes to whitelist request urls (overrides blacklist)")
    blacklistFilename := flag.String("bl", "blacklist.txt", "file of regexes to blacklistlist request urls")

    flag.Parse()
    whiteList, wlErr := getRegexlist(*whitelistFilename)
    if wlErr != nil {
        log.Fatalf("Could not load whitelist. Error: %s", wlErr)
    }
    blackList, blErr := getRegexlist(*blacklistFilename)
    if blErr != nil {
        log.Fatalf("Could not load blacklist. Error: %s", blErr)
    }

    // Start longpoll subscription manager
    eventChan, eventAjaxHandler := longpolling.StartLongpollManager()
    // Create and start control server for controlling proxy behavior
    ctlServer := NewControlServer(controlPort, eventAjaxHandler)
    ctlServer.Serve()

    // Create and start our content blocking proxy:
    proxy := goproxy.NewProxyHttpServer()
    proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
    proxy.OnRequest().DoFunc(func (req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
        // Prevent upgrades to https so we can easily see everything as plain
        if req.URL.Scheme == "https" {
            req.URL.Scheme = "http"
        }
        urlString := req.URL.String()
        for _, w := range whiteList {
            if w.MatchString(urlString) {
                log.Printf("WHITELISTED:  %s\n", req.URL)
                notifyEvent("Allowed", req, eventChan)
                return req, nil
            }
        }
        // See if we're manually allowing this page thru
        if (strings.HasSuffix(urlString, proxyExceptionString)) {
            urlString := urlString[:len(urlString) - len(proxyExceptionString)]
            u, uErr := url.Parse(urlString)
            if uErr == nil {
                req.URL = u
                log.Printf("MANUALLY ALLOWED: %s\n", req.URL)
                notifyEvent("Manually Allowed", req, eventChan)
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
                notifyEvent("Blocked", req, eventChan)
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
                        </html>`, req.URL, req.URL, proxyExceptionString))
            }
        }
        log.Printf("NOT MATCHED: (allow by default) %s\n", req.URL)
        notifyEvent("Not matched, default allowed", req, eventChan)
        return req, nil
    })

    proxy.OnResponse(goproxy_html.IsHtml).Do(goproxy_html.HandleString(
        func(s string, ctx *goproxy.ProxyCtx) string {
            if (strings.HasPrefix(ctx.Req.URL.Host, "127.0.0.1") || strings.HasPrefix(ctx.Req.URL.Host, "localhost")) {
                // Don't inject on our own content.
                // TODO: move this logic next to IsHtml so this func
                // never called?
                return s;
            }
            match := startBodyTagMatcher.FindIndex([]byte(s))
            if match != nil && len(match) >= 2 {
                // TODO: make this more efficient by using a stream or some sort
                // of stringbuilder like thing that doesn't require mashing
                // giant strings together.
                return s[:match[1]] +
                    getParentControlScript() +
                    "<div id=\"proxyblock-controls\" style=\"position: fixed; height: 42px; width: 222px !important; top: 4px; right: 8px; z-index: 99999999;\">" +
                    "<iframe scrolling=\"no\" style=\"overflow: hidden; background-color: #FFFFFF; border: 2px solid black; width: 100%; height: 100%;\" " +
                    "src=\"http://127.0.0.1:" + controlPort + "/page-menu?page=" + ctx.Req.URL.String()  + "\"></iframe>" +
                    "</div>" +
                    s[match[1]:]
            } else {
                log.Printf("WARNING: No starting body tag found, must not be html, no injection.")
                return s
            }
        }))

    proxy.Verbose = *verbose
    // Start proxy (this call is blocking)
    log.Fatal(http.ListenAndServe(*addr, proxy))

}

func getRegexlist(filename string) ([]*regexp.Regexp,  error) {
    file, err := os.Open(filename)
    if err != nil {
        log.Fatalf("Error opening %s: %q", filename, err)
    }
    defer file.Close()
    scanner := bufio.NewScanner(file)
    var list []*regexp.Regexp = make([]*regexp.Regexp, 0)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        // ignore blank/whitespace lines and comments
        if len(line) > 0 && !strings.HasPrefix(line, "#") {
            // add ignore case option to regex and compile it
            if r, err := regexp.Compile("(?i)" + line) ; err == nil {
                list = append(list, r)
            } else {
                log.Fatalf("Invalid pattern: %q", err)
            }
        }
    }
    if err := scanner.Err(); err != nil {
        log.Fatalf("Error reading %s: %q", filename, err)
    }
    return list, nil
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
            alert(e.origin);
            alert(e.data);
        }, false);
    </script>
    `
}

func pageMenuHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <script src="http://code.jquery.com/jquery-1.11.3.min.js%s"></script>
    <style>
        body {
            color: #000000;
            background-color: #EEEEEE;
        }
        #control-wrapper {
            display: block;
            padding: 0;
            margin: 0;
            width: 100%%;
        }
        #page-controls {
            display: block;
            clear: both;
            background-color: transparent;
            margin: 0 0 10px 0;
            padding: 0;
            width: 220px;
        }
        .control-item {
            display: inline-block;
            padding: 2px 4px;
            margin: 0;
            font-size: 14px;
            font-weight: bold;
            cursor: pointer;
            width: 26px;
            text-align: center;
            border: 2px solid transparent;
        }
        .control-item:hover {
            border: 2px solid black;
        }

        #stat-num-allow {
            background-color: #77FF77;
        }
        #stat-num-block {
            background-color: #FF7777;
        }
        #stat-num-manual {
            background-color: #FFFF77;
        }
        #move-controls {
            background-color: #BBBBBB;
        }
        #toggle-details {
            background-color: #AABBFF;
        }
        #info {
            font-size: 12px;
            font-weight: normal;
            color: #000000;
            padding: 0;
            margin: 0;
        }
    </style>
</head><body>
    <div id="control-wrapper">
        <div id="page-controls">
            <div id="stat-num-allow" class="control-item">0</div>
            <div id="stat-num-block" class="control-item">0</div>
            <div id="stat-num-manual" class="control-item">0</div>
            <div id="move-controls" class="control-item">&#x25BC;</div>
            <div id="toggle-details" class="control-item">+</div>
        </div>
    </div>
    <h3 id="info"></h3>
    <table border=1>
      <tr>
        <th>Requests</th>
      </tr>
      <tr id="stuff-happening">
      </tr>
    </table>
    </div>
    <script type="text/javascript">

    // Start checking events from a few seconds ago in case our iframe didn't
    // load right away due to other js on parent page being slow
    var sinceTime = ISODateString( new Date(Date.now() - 10000) );

    var stats = {
        blocked: 0,
        allowed: 0,
        manual: 0
    };

    var controlState = {
        upTop: true,
        expanded: false
    };

    (function poll() {
        var category = location.search;
        var exceptionString = "%s";
        if (category.length > 6) {
            category = category.slice(6, category.length);
        }
        // get rid of proxy exception string so we don't break our notification
        // subscription category
        if (stringEndsWith(category, exceptionString)) {
            category = category.slice(0, exceptionString.length);
        }
        $('#info').text(category);
        var timeout = 15;

        var optionalSince = "";
        if (sinceTime) {
            optionalSince = "&since_time=" + sinceTime;
        }
        var pollUrl = "http://127.0.0.1:%s/events?timeout=" + timeout + "&category=" + category + optionalSince;
        console.log(pollUrl);
        $.ajax({ url: pollUrl,
            success: function(data) {
                var receivedTime = (new Date()).toISOString();
                if (data && data.events && data.events.length > 0) {
                    // Events are most recent first, so insertBefore from end of array
                    // to keep latest event on top
                    for (var i = data.events.length - 1; i >= 0 ; i--) {
                        tally(data.events[i]);
                        $("#stuff-happening").append(getFormattedEvent(data.events[i], receivedTime));
                        sinceTime = data.events[i].timestamp;
                    }
                }
                if (data && data.events && data.events.length == 0) {
                    console.log("Empty events, that's weird!")
                }
                if (data && data.timeout) {
                    console.log("No events, checking again.");
                }
                if (data && data.error) {
                    console.log("Error response: " + data.error);
                    console.log("Trying again shortly...")
                    sleep(1000);
                }
            }, dataType: "json",
        error: function (data) {
            console.log(data);
            console.log("Error in ajax request--trying again shortly...");
            //sleep(3000);
        },
        complete: poll
        });
    })();


    function stringEndsWith(str, suffix) {
        return str.indexOf(suffix, str.length - suffix.length) !== -1;
    };

    function getFormattedEvent(event) {
      return "<tr class='event-item'>" +
        "<td>" + event.data + "</td>" +
        "</tr>";
    };

    function tally(event) {
        if (!event || !event.data) {
            return;
        }
        if (event.data.slice(0,1) == 'A') {
            stats.allowed += 1;
            $('#stat-num-allow').html(stats.allowed);
        } else if (event.data.slice(0,1) == 'B') {
            stats.blocked += 1;
            $('#stat-num-block').html(stats.blocked);
        } else if (event.data.slice(0,1) == 'M') {
            stats.manual += 1;
            $('#stat-num-manual').html(stats.manual);
        } else {
            // else unknown event :(
            return;
        }
    };

    function sleep(milliseconds) {
      var start = new Date().getTime();
      for (var i = 0; i < 1e7; i++) {
        if ((new Date().getTime() - start) > milliseconds){
          break;
        }
      }
    };

    /* use a function for the exact format desired... */
    function ISODateString(d){
        function pad(n){return n<10 ? '0'+n : n}
        return d.getUTCFullYear()+'-'
           + pad(d.getUTCMonth()+1)+'-'
           + pad(d.getUTCDate())+'T'
           + pad(d.getUTCHours())+':'
           + pad(d.getUTCMinutes())+':'
           + pad(d.getUTCSeconds())+'Z'
    };

    $("#toggle-details").click(function(event) {
        controlState.expanded = !controlState.expanded;
        window.parent.postMessage({expanded: controlState.expanded}, "*");
    });

    $("#move-controls").click(function(event) {
        controlState.upTop = !controlState.upTop;
        window.parent.postMessage({upTop: controlState.upTop}, "*");
    });

    </script>
</body>
</html>`, proxyExceptionString, proxyExceptionString, controlPort)
}

