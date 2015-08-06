package main

import (
    "github.com/elazarl/goproxy"
    "log"
    "flag"
    "bufio"
    "os"
    "fmt"
    "strings"
    "regexp"
    "net/http"
    "net/url"
)

// TODO: Try and get HTTPS MITM working so we can also block javascript/ads on
// https sites

// TODO: remember exceptions, allow option to except anythign on a given
// domain or subdomain.domain

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
    proxyExceptionString := "LOL-WHUT-JUST-DOIT-DOOD"
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
    proxy.Verbose = *verbose
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

