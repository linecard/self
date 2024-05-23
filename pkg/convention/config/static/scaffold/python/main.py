import http
import socketserver
import logging

PORT = 8000

class GetHandler(
        http.server.SimpleHTTPRequestHandler
        ):

    def do_GET(self):
        logging.error(self.headers)
        http.server.SimpleHTTPRequestHandler.do_GET(self)


Handler = GetHandler
httpd = socketserver.TCPServer(("", 8080), Handler)
httpd.serve_forever()