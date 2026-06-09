from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import urlparse, parse_qs
import sys
import time

port = int(sys.argv[1])

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        params = parse_qs(urlparse(self.path).query)
        delay = int(params.get("delay", ["0"])[0])
        if delay > 0:
            time.sleep(delay)
        self.send_response(200)
        self.end_headers()
        self.wfile.write(f"backend:{port} path:{self.path}\n".encode())

    def log_message(self, fmt, *args):
        print(f"[:{port}] {fmt % args}")

HTTPServer(("", port), Handler).serve_forever()