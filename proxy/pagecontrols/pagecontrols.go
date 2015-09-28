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
            margin: 0;
        }
        #toggle-details {
            background-color: #AABBFF;
            float: right;
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
        td.request-status.status-blocked {
            color: #000000;
            background-color: #FF8888;
        }
        td.request-status.status-manual {
            color: #000000;
            background-color: #FFFF88;
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
            <div id="move-controls" class="control-item">&#x25BC;</div>
            <div id="toggle-details" class="control-item">+</div>
        </div>
    </div>
    <br />
    <h3 id="info"></h3>
    <table id="event-table" border=0>
      <tr>
        <th>Status</th>
        <th>Time</th>
        <th id="requests-title">Requests</th>
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
        var timeout = 15;
        var optionalSince = "";
        if (sinceTime) {
            optionalSince = "&since_time=" + sinceTime;
        }
        var pollUrl = "http://127.0.0.1:%s/events?timeout=" + timeout + "&category=" + category + optionalSince;
        $.ajax({ url: pollUrl,
            success: function(data) {
                var receivedTime = (new Date()).toISOString();
                if (data && data.events && data.events.length > 0) {
                    for (var i = data.events.length - 1; i >= 0 ; i--) {
                        tally(data.events[i]);
                        $("#stuff-happening").before(getFormattedEvent(data.events[i], receivedTime));
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
            console.log("Error in ajax request--trying again shortly...");
        },
        complete: poll
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
            var rowClass = "status-unknown";
            if (event.data.slice(0,1) == 'A') {
                status = "<td class=\"request-status status-allowed\">Allowed</td>";
                rowClass = "status-allowed";
            } else if (event.data.slice(0,1) == 'B') {
                status = "<td class=\"request-status status-blocked\">Blocked</td>";
                rowClass = "status-blocked";
            } else if (event.data.slice(0,1) == 'M') {
                status = "<td class=\"request-status status-manual\">Manual</td>";
                rowClass = "status-manual";
            }
            var d = new Date(event.timestamp);
            var t = d.toLocaleTimeString();
            return "<tr class='event-item " + rowClass + "'>" + status +
                "<td>" + t.slice(0, t.length - 3) + "</td>" +
                "<td class='request-url'><span class='url'>" + url + "</span>" +
                "<div class=\"details-wrapper\"></div>" +
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
        if (controlState.expanded) {
            $(this).html("_");
            setTimeout(function () {
                $("#open-settings").addClass("showme");
            }, 200);
        } else {
            $(this).html("+");
            $("#open-settings").removeClass("showme");
        }
        window.parent.postMessage({expanded: controlState.expanded}, "*");
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
    });

    $("#stat-num-block").click(function(event) {
        $("#stat-num-allow").removeClass("activated");
        $("#stat-num-manual").removeClass("activated");
        $("#event-table").removeClass("status-allowed");
        $("#event-table").removeClass("status-manual");
        $("#event-table").toggleClass("status-blocked");
        $(this).toggleClass("activated");
        updateRequestColTitle();
    });

    $("#stat-num-manual").click(function(event) {
        $("#stat-num-allow").removeClass("activated");
        $("#stat-num-block").removeClass("activated");
        $("#event-table").removeClass("status-allowed");
        $("#event-table").removeClass("status-blocked");
        $("#event-table").toggleClass("status-manual");
        $(this).toggleClass("activated");
        updateRequestColTitle();
    });

    $(document).on("click", "tr.event-item", function(event){
        if ($(this).hasClass("details")) {
            $(this).removeClass("details");
            $("div.details-wrapper", $(this)).html("");
        } else {
            $(this).addClass("details");
            $("div.details-wrapper", $(this)).html("<h3>TODO - exception once versus always click</h3>if allowed option to block, or vice versa,  option to do either if manually allowed.  Also link to open new tab showing what rule made it allowed/blocked.");
        }
    });
    </script>
</body>
</html>`,
    vars.ProxyExceptionString,
    vars.ProxyExceptionString,
    vars.ControlPort)
}
