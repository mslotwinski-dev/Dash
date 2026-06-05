import http.server
import socketserver
import threading

class Handler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.send_header("Content-type", "text/html")
        self.end_headers()
        msg = f"Hello from server on port {self.server.server_address[1]}!"
        self.wfile.write(msg.encode())


for port in [3000, 3001]:
    server = socketserver.ThreadingTCPServer(("", port), Handler)
    threading.Thread(
        target=server.serve_forever,
        daemon=True
    ).start()

print("Servers running")

threading.Event().wait()