# ProxyBlock
A proxy that uses a whitelist/blacklist to block unwanted web content like ads, third party javascript, and other cruft.

Websites should load much faster despite being run through a proxy since all the ads and other javascript/tracking cruft typically gets blocked out.  Many sites may have some missing styles or features, but with enough configuring it can make for a much snappier web experience.

## Running
If you have a go workspace set up, simply build and run out of the box:
```
go build proxyblock.go
./proxyblock
```
Then just configure your browser to go thru the proxy.

## Configuring
You can modify the whitelist.txt and blacklist.txt files.
These files contain a list of regexes (and optional comments that must start at
the beginning of a line).  URLs that match whitelist patterns will be allowed through
while URLs that match blacklist patterns will be blocked.  If a URL matches neither, it is allowed by default.

You can also manually allow a page by clicking the continue link on the proxy block response webpage.

