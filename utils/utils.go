package utils

// Common utility functions

import (
	"bufio"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

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

// adapted from:
// http://stackoverflow.com/questions/13294649/how-to-parse-a-milliseconds-since-epoch-timestamp-string-in-go
func MillisecondStringToTime(ms string) (time.Time, error) {
	msInt, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(0, msInt*int64(time.Millisecond)), nil
}

// adapted from:
// http://stackoverflow.com/questions/24122821/go-golang-time-now-unixnano-convert-to-milliseconds
func TimeToEpochMilliseconds(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}
