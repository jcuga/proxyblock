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
    "math/rand"

    "github.com/jcuga/proxyblock/longpolling"
)

var (
    endBodyTagMatcher = regexp.MustCompile(`(?i:</body>)`)
)

type HTTPServer struct {
    port string
    https *http.Server
}

func (s *HTTPServer) Serve() {
    go s.https.ListenAndServe()
}

func pageMenuHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "TESTING 123")
}

func NewControlServer(port string ) (*HTTPServer) {
    s := &HTTPServer{port,&http.Server{Addr: "127.0.0.1:" + port, Handler: nil } }
    mux := http.NewServeMux()
    mux.HandleFunc("/page-menu", pageMenuHandler)
    s.https.Handler = mux
    return s
}

func main() {
    controlPort := "8380"
    longpollPort := "8280"

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
    // TODO: make exception a uuid generated on each start
    proxyExceptionString := "LOL-WHUT-JUST-DOIT-DOOD"
    proxy := goproxy.NewProxyHttpServer()
    proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
    proxy.OnRequest().DoFunc(func (req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
        // Prevent upgrades to https so we can easily see everything as plain
        if req.URL.Scheme == "https" {
            req.URL.Scheme = "http"
        }


        // TODO: remove once done:
        if val := req.Header.Get("Referer") ; len(val) > 0 {
            log.Println("Referer: " + val)
        } else {
            log.Println("No Referer")
        }

        urlString := req.URL.String()
        for _, w := range whiteList {
            if w.MatchString(urlString) {
                log.Printf("WHITELISTED:  %s\n", req.URL)
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
        return req, nil
    })

    proxy.OnResponse(goproxy_html.IsHtml).Do(goproxy_html.HandleString(
        func(s string, ctx *goproxy.ProxyCtx) string {
            match := endBodyTagMatcher.FindIndex([]byte(s))
            if match != nil && len(match) == 2 {
                // TODO: make this more efficient by using a stream or some sort
                // of stringbuilder like thing that doesn't require mashing
                // giant strings together.
                return s[:match[0]] +
                    "<iframe style=\"position:fixed; height: 50px; " +
                    "width: 220px; top: 4px; right: 100px; " +
                    "z-index: 99999999;\" " +
                    "src=\"http://127.0.0.1:" + controlPort + "/page-menu\"></iframe>" +
                    s[match[0]:]
            } else {
                log.Printf("No closing body tag found, must not be html, no injection.")
                return s
            }
        }))
    proxy.Verbose = *verbose
    // Start proxy
    go func () {
        log.Fatal(http.ListenAndServe(*addr, proxy))
    }()

    // Create and start control server for controlling proxy behavior
    ctlServer := NewControlServer(controlPort)
    ctlServer.Serve()

    // Create and start longpoll server for serving proxy events
    eventChan := longpolling.StartLongpollServer(longpollPort)


    // Dummy code to exercise longpoll server.  TODO: remove once replaced with real events
    categories := []string{"apple", "banana", "pear", "orange"}
    data := []string{"hi mom", "asdf123", "this is some data",
        "datar pl33ze!!!", "0101010101000101110100101010100",
        "Foobar widgets", "nuggets", "cows"}
    // Send events with random category/data values
    for {
        select {
        case <-time.After(3000 * time.Millisecond):
            event := longpolling.Event{time.Now(), categories[rand.Intn(len(categories))], data[rand.Intn(len(data))]}
            eventChan <- event
        }
    }
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

