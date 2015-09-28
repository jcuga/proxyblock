package utils

// Common utility functions

import (
    "log"
    "bufio"
    "os"
    "strings"
    "regexp"

    "github.com/jcuga/proxyblock/proxy/vars"
)

// Parse a file of regular expressions, ignoring comments/whitespace
func GetRegexlist(filename string) ([]*regexp.Regexp, error) {
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
            if r, err := regexp.Compile("(?i)" + line); err == nil {
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

// Since our event subscriptions (longpoll) are based on a 'category' which is
// the URL/referer, when we add a proxy exception string to manually bypass
// content blocking, the string on the end of the URL will cause a mismatch and
// we'll never get our content accepted/blocked notifications
func StripProxyExceptionStringFromUrl(url string) string {
    if strings.HasSuffix(url, vars.ProxyExceptionString) {
        return url[:len(url)-len(vars.ProxyExceptionString)]
    } else {
        return url
    }
}
