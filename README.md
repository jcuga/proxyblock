# ProxyBlock
A proxy that uses a whitelist/blacklist to block unwanted web content like ads, third party javascript, and other cruft.

Websites should load much faster despite being run through a proxy since all the ads and other javascript/tracking cruft typically gets blocked out.  Many sites may have some missing styles or features, but with enough configuring it can make for a much snappier web experience.

## Running
If you have a go workspace set up, simply build and run out of the box:
```
go get ./...
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

## Brief Overview
![screenshot 0](https://raw.githubusercontent.com/jcuga/proxyblock/master/demo-screenshots/demo-screenshot-0.png)
![screenshot 1](https://raw.githubusercontent.com/jcuga/proxyblock/master/demo-screenshots/demo-screenshot-1.png)
![screenshot 2](https://raw.githubusercontent.com/jcuga/proxyblock/master/demo-screenshots/demo-screenshot-2.png)
![screenshot 3](https://raw.githubusercontent.com/jcuga/proxyblock/master/demo-screenshots/demo-screenshot-3.png)
![screenshot 4](https://raw.githubusercontent.com/jcuga/proxyblock/master/demo-screenshots/demo-screenshot-4.png)
![screenshot 5](https://raw.githubusercontent.com/jcuga/proxyblock/master/demo-screenshots/demo-screenshot-5.png)


## TODO
* HTTPS/Man-in-the-Middle proxying to control HTTPS content
* better injected UI
* interactive whitelist/blacklist regexp tweaking
* persisting exceptions between runs
