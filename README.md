# ProxyBlock
A proxy that uses a whitelist/blacklist to block unwanted web content like ads, third party javascript, and other cruft.
The proxy injects an iframe into web content to allow users to fine tune the proxy.

This is a different take on blocking ads/tracking-javascript that does everything
via a proxy instead of having a browser plugin.  This is an interesting proof of
concept that I had a lot of fun slapping together.

Many websites may load much faster despite being run through a proxy since all the ads and other javascript/tracking cruft typically gets blocked out.  Many sites may have some missing styles or features, but with enough configuring it can make for a much snappier web experience.

## Current State
This code has reached full prototype status, it essentially does what my
original idea was.  I've used it on a daily basis to browse my usual news
websites (with more detailed configurations than the whitelist/blacklist checked
in here).  Overall it generally works as expected.  If HTTPS support is added,
the UI given a makeover, and I add interactive whitelist/blacklist pattern
editing from the controls, then I think this could be a pretty handy tool.

## Running
Check out the latest release at:  https://github.com/jcuga/proxyblock/releases

Or, if you have a go workspace set up, simply build and run out of the box:
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

## How to Use
Once you've set your browser/OS to use the proxy you'll get a block page for any
content that is blacklisted per your ```blacklist.txt``` configuration file.
You can optionally add the page URL to the whitelist, or visit it just one time
only:
![screenshot 0](https://raw.githubusercontent.com/jcuga/proxyblock/master/demo-screenshots/demo-screenshot-0.png)

When you're viewing webpages that aren't blocked, you'll see an injected
"page control" that shows how many requests for that page were allowed/blocked/
manually allowed.  Notice that until you configure the proxy a bit more, many
webpages you visit may fail to load stylesheet or certain javascript due to
those pages' urls being blacklisted.
![screenshot 1](https://raw.githubusercontent.com/jcuga/proxyblock/master/demo-screenshots/demo-screenshot-1.png)

You can click to open the page controls and see more information about what
was allowed/blocked.
![screenshot 2](https://raw.githubusercontent.com/jcuga/proxyblock/master/demo-screenshots/demo-screenshot-2.png)

You can manually click a row to unblock/block
future requests for given URLs.  Then when you visit the page again (just hit
reload), you'll get the content.
![screenshot 3](https://raw.githubusercontent.com/jcuga/proxyblock/master/demo-screenshots/demo-screenshot-3.png)

By the way, you can move the page controls by clicking the up/down arrow.
The proxy doesn't remember your preference between page loads though--that's one
of the many TODOs.
![screenshot 4](https://raw.githubusercontent.com/jcuga/proxyblock/master/demo-screenshots/demo-screenshot-4.png)


## TODO
* Change/configure order of whitelist/blacklist rule application, currently the
    whitelist is applied first, then the blacklist.
* HTTPS/Man-in-the-Middle proxying to control HTTPS content
* better injected UI
* wider browser support/testing for UI/behavior.
* interactive whitelist/blacklist regexp tweaking
* persisting exceptions between run
* thread safety, currently there are goroutines all referencing a map,
    that's easy to fix...