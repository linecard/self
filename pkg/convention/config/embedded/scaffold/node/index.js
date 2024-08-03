const http = require('http');

const PORT = 8080;

const server = http.createServer((req, res) => {
    if (req.url === '/healthz' && req.method === 'GET') {
        res.writeHead(200, { 'Content-Type': 'text/plain' });
        res.end('OK');
    } else if (req.url === '/' && req.method === 'GET') {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify(req.headers, null, 2));
    } else {
        res.writeHead(404, { 'Content-Type': 'text/plain' });
        res.end(`${req.method} ${req.url} not found`);
    }
});

server.listen(PORT, '0.0.0.0', () => {
    console.log(`Server running at http://0.0.0.0:${PORT}/`);
});
