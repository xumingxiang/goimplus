(function(win) {
  
    var Client = function(options) {
        var MAX_CONNECT_TIMES = 10;
        var DELAY = 15000;
        this.options = options || {};
        this.createConnect(MAX_CONNECT_TIMES, DELAY);
    }

    Client.prototype.createConnect = function(max, delay) {
        var self = this;
        if (max === 0) {
            return;
        }
        connect();

        var heartbeatInterval;
        function connect() {
            var ws = new WebSocket('ws://192.168.4.7:8090/sub');
            ws.onopen = function() {
                auth();
            }

            ws.onmessage = function(evt) {
                var data = evt.data;
                  messageReceived( data);
            }

            ws.onclose = function() {
                if (heartbeatInterval) clearInterval(heartbeatInterval);
                setTimeout(reConnect, delay);
            }

            function heartbeat() {
            
                ws.send("headerBuf");
                console.log("send: heartbeat");
            }

            function auth() {
                var token = "{\"userId\":1000,\"roomId\":16912}" // 授权的格式
                ws.send(token);
            }

            function messageReceived(body) {
                var notify = self.options.notify;
                if(notify) {
                    notify(body);
                }
                console.log("messageReceived:","body=" + body);
            }
        }

        function reConnect() {
            self.createConnect(--max, delay * 2);
        }
    }

    win['MyClient'] = Client;
})(window);
