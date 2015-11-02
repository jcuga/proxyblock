package pagecontrols

// Every page that gets proxied has an iframe injected with DOM/UI to see what
// get's blocked/allowed as well as allowing the user to control the browsing
// experience.

import (
	"fmt"
	"net/http"

	"github.com/jcuga/proxyblock/proxy/vars"
)

const (
	ProxyPageControlsUrl = "/page-controls"
)

// Get the URL to our proxy page controls UI
// Takes the original content url that we're proxying.
func GetPageControlsUrl(url string) string {
	return fmt.Sprintf("%s?page=%s", ProxyPageControlsUrl, url)
}

// Serves our proxy content page controls.  This is loaded in an iframe that
// gets injected in every page we proxy.  This shows proxy stats (blocked,
// allowed, manually allowed) as well as listing all requests made from that
// page and (TODO) links to block/unblock those requests in the future.
func PageControlsHandler(w http.ResponseWriter, r *http.Request) {
	// NOTE: '%' characters must be escaped as '%%'
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <script src="http://code.jquery.com/jquery-1.11.3.min.js%s"></script>
    <style>
        body {
            color: #000000;
            background-color: #EEEEEE;
            font-family: monospace;
            font-size: 12px;
            overflow: auto;
        }
        .event-item .details-wrapper {
            display: none;
        }

        .event-item.details .details-wrapper {
            display: block;
        }

        .item-control-links .add-wl {
            padding: 6px;
            background-color: #00FF00;
            color: #000000;
            border: 1px solid #000000;
            display: inline-block;
        }
        .item-control-links .add-wl:hover {
            color: #FFFFFF;
            border: 1px solid #FFFFFF;
        }
        .item-control-links .add-bl {
            padding: 6px;
            background-color: #FF0000;
            color: #000000;
            border: 1px solid #000000;
            display: inline-block;
        }
        .item-control-links .add-bl:hover {
            color: #FFFFFF;
            border: 1px solid #FFFFFF;
        }
        #page-controls {
            display: block;
            clear: both;
            background-color: transparent;
            margin: 0 0 10px 0;
            padding: 0;
            width: 100%%;
            height: 10px;
        }
        .control-item {
            display: inline-block;
            padding: 2px 4px;
            margin: 0 4px 0 0;
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
            float: left;
        }
        #stat-num-block {
            background-color: #FF7777;
            float: left;
        }
        #stat-num-manual {
            background-color: #FFFF77;
            float: left;
            margin-right: 6px;
        }
        #move-controls {
            background-color: #BBBBBB;
            float: right;
        }
        #toggle-details {
            background-color: #AABBFF;
            float: right;
            margin: 0;
        }
        #info {
            font-size: 14px;
            font-weight: normal;
            color: #000000;
            padding: 0;
            margin: 0 0 0 4px;
        }
        table {
            border: 0;
            margin: 6px 0 0 0;
            padding: 0;
        }
        th {
            text-align: left;
            padding: 3px 4px;
        }
        tr {
            padding: 0;
            margin: 0;
            background-color: #DDDDFF;
            cursor: default;
        }
        tr:nth-child(even) {
            background-color: #EEEEFF;
        }
        tr:hover {
            background-color: #FFFFCC;
        }
        tr.event-item {
            cursor: pointer;
        }
        td {
            padding: 3px 4px;
            text-align: left;
            vertical-align: top;
            border-top: 1px solid black;
        }
        td.request-status.status-allowed {
            color: #000000;
            background-color: #88FF88;
        }
        td.request-status.status-allowed.now-blacklisted {
            background-color: #FFAA88;
        }
        td.request-status.status-blocked {
            color: #000000;
            background-color: #FF8888;
        }
        td.request-status.status-blocked.now-whitelisted {
            background-color: #BBEE88;
        }
        td.request-status.status-manual {
            color: #000000;
            background-color: #FFFF88;
        }
        td.request-status.status-manual.now-whitelisted {
            background-color: #BBEE88;
        }
        td.request-status.status-manual.now-blacklisted {
            background-color: #FFAA88;
        }
        .control-item.activated {
            border: 2px solid #0000FF;
        }
        #open-settings {
            color: #000000;
            display: none;
            width: 67px;
            margin: 0;
            background-color: #44DDFF;
        }
        #open-settings.showme {
            display: inline-block;
        }
        #event-table.status-blocked tr.status-allowed, #event-table.status-blocked tr.status-manual,
        #event-table.status-allowed tr.status-blocked, #event-table.status-allowed tr.status-manual,
        #event-table.status-manual tr.status-blocked, #event-table.status-manual tr.status-allowed {
            display: none;
        }
    </style>
</head><body>
    <div id="control-wrapper">
        <div id="page-controls">
            <div id="stat-num-allow" class="control-item">0</div>
            <div id="stat-num-block" class="control-item">0</div>
            <div id="stat-num-manual" class="control-item">0</div>
            <a href="/proxy-settings" target="_open_proxy_settings"><div id="open-settings" class="control-item">Settings</div></a>
            <div id="toggle-details" class="control-item">+</div>
            <div id="move-controls" class="control-item">&#x25BC;</div>
        </div>
    </div>
    <br />
    <h3 id="info"></h3>
    <table id="event-table" border=0>
      <tr>
        <th>Status</th>
        <th>Time</th>
        <th>Type</th>
        <th id="requests-title">Requests</th>
      </tr>
      <tr id="stuff-happening">
      </tr>
    </table>
    </div>
    <script type="text/javascript">

    // Start checking events from a few (10) seconds ago in case our iframe
    // didn't load right away due to other js on parent page being slow.
    // Note: using time as milliseconds since epoch instead of some string
    // timestamp like earlier which caused internationalization issues
    var sinceTime = (new Date(Date.now() - 10000)).getTime();

    var stats = {
        blocked: 0,
        allowed: 0,
        manual: 0
    };

    var controlState = {
        upTop: true,
        expanded: false
    };

    function toggleDetailsView(detailButton) {
        var detailButton = $("#toggle-details");
        controlState.expanded = !controlState.expanded;
        if (controlState.expanded) {
            detailButton.html("_");
            setTimeout(function () {
                $("#open-settings").addClass("showme");
            }, 200);
        } else {
            detailButton.html("+");
            $("#open-settings").removeClass("showme");
        }
        window.parent.postMessage({expanded: controlState.expanded}, "*");
    }

    // for browsers that don't have console
    if(typeof window.console == 'undefined') { window.console = {log: function (msg) {} }; }

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
        $('#info').attr('alt', category);
        var timeout = 15;  // in seconds
        var optionalSince = "";
        if (sinceTime) {
            optionalSince = "&since_time=" + sinceTime;
        }
        var pollUrl = "http://127.0.0.1:%s/events?timeout=" + timeout + "&category=" + category + optionalSince;
        // how long to wait before starting next longpoll request in each case:
        var successDelay = 10;  // 10 ms
        var errorDelay = 3000;  // 3 sec
        $.ajax({ url: pollUrl,
            success: function(data) {
                if (data && data.events && data.events.length > 0) {
                    // got events, process them
                    for (var i = 0; i < data.events.length; i++) {
                        tally(data.events[i]);
                        $("#stuff-happening").before(getFormattedEvent(data.events[i]));
                        sinceTime = data.events[i].timestamp;
                    }
                    // success!  start next longpoll
                    setTimeout(poll, successDelay);
                    return;
                }
                if (data && data.events && data.events.length == 0) {
                    console.log("Empty events, that's weird!")
                    // should get a timeout response, not an empty event array
                    // if no events during longpoll window.  so this is weird
                    setTimeout(poll, errorDelay);
                    return;
                }
                if (data && data.timeout) {
                    console.log("No events, checking again.");
                    // no events within timeout window, start another longpoll:
                    setTimeout(poll, successDelay);
                    return;
                }
                if (data && data.error) {
                    console.log("Error response: " + data.error);
                    console.log("Trying again shortly...")
                    setTimeout(poll, errorDelay);
                    return;
                }
                console.log("Didn't get expected event data, try again shortly...");
                setTimeout(poll, errorDelay);
            }, dataType: "json",
        error: function (data) {
            console.log("Error in ajax request--trying again shortly...");
            setTimeout(poll, 3000);  // 3s
        }
        });
    })();


    function stringEndsWith(str, suffix) {
        return str.indexOf(suffix, str.length - suffix.length) !== -1;
    };

    function getFormattedEvent(event) {
        if (!event && !event.data) {
            return "";
        }
        var i = event.data.indexOf(": ");
        if (i > 0) {
            var status = "<td class=\"request-status status-unknown\">???</td>";
            var url =  event.data.slice(i + 2, event.data.length);
            var controlLinks = "<p class=\"item-control-links\">";
            var rowClass = "status-unknown";
            if (event.data.slice(0,1) == 'A') {
                status = "<td class=\"request-status status-allowed\">Allowed</td>";
                rowClass = "status-allowed";
                controlLinks += "<span class=\"add-bl\">Blacklist URL</span>";
            } else if (event.data.slice(0,1) == 'B') {
                status = "<td class=\"request-status status-blocked\">Blocked</td>";
                rowClass = "status-blocked";
                controlLinks += "<span class=\"add-wl\">Whitelist URL</span>";
            } else if (event.data.slice(0,1) == 'M') {
                status = "<td class=\"request-status status-manual\">Manual</td>";
                rowClass = "status-manual";
                controlLinks += "<span class=\"add-wl\">Whitelist URL</span>";
                controlLinks += "<span class=\"add-bl\">Blacklist URL</span>";
            }
            controlLinks += "</p>";
            var d = new Date(event.timestamp);
            var t = d.toLocaleTimeString();
            return "<tr class='event-item " + rowClass + "'>" + status +
                "<td>" + t.slice(0, t.length - 3) + "</td>" +
                "<td>" + guessContent(url) + "</td>" +
                "<td class='request-url'><span class='url'>" + url + "</span>" +
                "<div class=\"details-wrapper\">" +
                controlLinks +
                "</div>" +
                "</td>" +
                "</tr>";
        }
        return "";
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

    var contentPatterns = {
        css: new RegExp("^.*css[^\./]*$"),
        jpg: new RegExp("^.*jpg[^\./]*$"),
        png: new RegExp("^.*png[^\./]*$"),
        gif: new RegExp("^.*gif[^\./]*$"),
        js: new RegExp("^.*js[^\./]*$"),
        html: new RegExp("^.*html[^\./]*$"),
        html2: new RegExp("^.*/$")
    };

    function guessContent(url) {
        if (contentPatterns.css.exec(url)) {
            return "CSS";
        }
        if (contentPatterns.jpg.exec(url)) {
            return "JPEG";
        }
        if (contentPatterns.png.exec(url)) {
            return "PNG";
        }
        if (contentPatterns.gif.exec(url)) {
            return "GIF";
        }
        if (contentPatterns.js.exec(url)) {
            return "JS";
        }
        if (contentPatterns.html.exec(url)) {
            return "HTML";
        }
        if (contentPatterns.html2.exec(url)) {
            return "HTML";
        }
        return "?";
    }

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
        toggleDetailsView();
    });

    $("#move-controls").click(function(event) {
        controlState.upTop = !controlState.upTop;
        if (controlState.upTop) {
            $(this).html("&#x25BC");
        } else {
            $(this).html("&#x25B2");
        }
        window.parent.postMessage({upTop: controlState.upTop}, "*");
    });

    function updateRequestColTitle() {
        var table = $("#event-table");
        if (table.hasClass("status-allowed")) {
            $("#requests-title").text("Requests (Allowed)");
        } else if (table.hasClass("status-blocked")) {
            $("#requests-title").text("Requests (Blocked)");
        } else if (table.hasClass("status-manual")) {
            $("#requests-title").text("Requests (Manual)");
        } else {
            $("#requests-title").text("Requests");
        }
    };

    $("#stat-num-allow").click(function(event) {
        $("#stat-num-manual").removeClass("activated");
        $("#stat-num-block").removeClass("activated");
        $("#event-table").removeClass("status-blocked");
        $("#event-table").removeClass("status-manual");
        $("#event-table").toggleClass("status-allowed");
        $(this).toggleClass("activated");
        updateRequestColTitle();
        // If we're not showing request details already, show them
        if (!controlState.expanded) {
            toggleDetailsView();
        }
    });

    $("#stat-num-block").click(function(event) {
        $("#stat-num-allow").removeClass("activated");
        $("#stat-num-manual").removeClass("activated");
        $("#event-table").removeClass("status-allowed");
        $("#event-table").removeClass("status-manual");
        $("#event-table").toggleClass("status-blocked");
        $(this).toggleClass("activated");
        updateRequestColTitle();
        // If we're not showing request details already, show them
        if (!controlState.expanded) {
            toggleDetailsView();
        }
    });

    $("#stat-num-manual").click(function(event) {
        $("#stat-num-allow").removeClass("activated");
        $("#stat-num-block").removeClass("activated");
        $("#event-table").removeClass("status-allowed");
        $("#event-table").removeClass("status-blocked");
        $("#event-table").toggleClass("status-manual");
        $(this).toggleClass("activated");
        updateRequestColTitle();
        // If we're not showing request details already, show them
        if (!controlState.expanded) {
            toggleDetailsView();
        }
    });

    $(document).on("click", "tr.event-item", function(event){
        $(this).toggleClass("details");
    });

    $(document).on("click", "tr.event-item .add-wl", function(event){
        event.stopPropagation();
        var item = $(this);
        var item_url = $(".url", item.parents(".request-url")).text() || "";
        if (!item.hasClass('clicked')) {
            item.addClass('clicked');
            item.text("Adding...");
            $.ajax({
                url: "/add-wl",
                type: "get",
                data:{url: item_url},
                success: function(response) {
                    item.text("Added to Whitelist.");
                    var statusArea = $(".request-status", item.parents(".event-item"));
                    if (statusArea) {
                        statusArea.html(statusArea.html() + "<br />Now Whitelisted");
                        statusArea.addClass("now-whitelisted");
                        item.parents(".event-item").removeClass("details");
                    }
                    // don't remove clicked class to prevent resends
                },
                error: function(xhr) {
                    item.text('ERROR adding to Whitelist.');
                    // let user try again.
                    item.removeClass('clicked');
                }
            });
        } else {
            // already clicked or succeeded, don't refire
            return;
        }
    });

    $(document).on("click", "tr.event-item .add-bl", function(event){
        event.stopPropagation();
        var item = $(this);
        var item_url = $(".url", item.parents(".request-url")).text() || "";
        if (!item.hasClass('clicked')) {
            item.addClass('clicked');
            item.text("Adding...");
            $.ajax({
                url: "/add-bl",
                type: "get",
                data:{url: item_url},
                success: function(response) {
                    item.text("Added to Blacklist.");
                    var statusArea = $(".request-status", item.parents(".event-item"));
                    if (statusArea) {
                        statusArea.html(statusArea.html() + "<br />Now Blacklisted");
                        statusArea.addClass("now-blacklisted");
                        item.parents(".event-item").removeClass("details");
                    }
                    // don't remove clicked class to prevent resends
                },
                error: function(xhr) {
                    item.text('ERROR adding to Blacklist.');
                    // let user try again.
                    item.removeClass('clicked');
                }
            });
        } else {
            // already clicked or succeeded, don't refire
            return;
        }
    });

        // Here "addEventListener" is for standards-compliant web browsers and "attachEvent" is for IE Browsers.
        var eventMethod = window.addEventListener ? "addEventListener" : "attachEvent";
        var eventer = window[eventMethod];
        // onmessage for attachEvent, message for addEventListener
        var messageEvent = eventMethod == "attachEvent" ? "onmessage" : "message";
        // Listen to message from parent window to know when to close detail view
        // this event is sent from the parent page to this iframe when the glass
        // overlay is clicked to dismiss the controlls.  this overlay is not
        // part of controls and thus we need to use events to do parent-to-iframe comms
         eventer(messageEvent, function (e) {
            if (e.data && e.data.closeDetails === true) {
                if (controlState.expanded) {
                    // close details
                    // TODO: put close details in func called by both spots
                    // instead of this copy n paste?
                    controlState.expanded = false;
                    $("#toggle-details").html("+");
                    $("#open-settings").removeClass("showme");
                    window.scrollTo(0, 0);
                }
            }
        }, false);

    </script>
</body>
</html>`,
		vars.ProxyExceptionString,
		vars.ProxyExceptionString,
		vars.ProxyControlPort)
}
