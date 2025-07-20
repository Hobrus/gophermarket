const http = require('http');

const delay = ms => new Promise(r => setTimeout(r, ms));

const server = http.createServer(async (req, res) => {
  if (req.method === 'GET' && /^\/api\/orders\/\d+$/.test(req.url)) {
    const num = req.url.split('/').pop();
    await delay(1000);
    res.setHeader('Content-Type', 'application/json');
    res.end(JSON.stringify({ order: num, status: 'PROCESSED', accrual: 1000 }));
  } else {
    res.statusCode = 204;
    res.end();
  }
});

server.listen(3000, '0.0.0.0');
