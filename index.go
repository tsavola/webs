// -*- javascript -*-

package webs

var indexHTML = []byte(`<!DOCTYPE html>
<html>
<body>
<script>
(function() {
    var ws;

    window.webs = {
        send: function(msg) {
	    ws.send(msg);
        }
    };

    function connect() {
        ws = new WebSocket(location.toString().replace(/^http/, "ws") + "io");

        ws.onclose = function() {
            ws.close();
            setTimeout(connect, 1000);
        };

        ws.onmessage = function(e) {
            eval(e.data);
        };
    }

    connect();
})();
</script>
</body>
</html>
`)
