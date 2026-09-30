const http = require("http");
const server = http.createServer((req, res) => {
  if (req.url === "/health") return res.end("ok");
  if (req.url === "/ready") return res.end("ready");
  if (req.url === "/metrics") return res.end("http_requests_total 0\n");
  res.statusCode = 404;
  res.end("not found");
});
server.listen(8080);
