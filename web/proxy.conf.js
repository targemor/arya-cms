// Forwards API calls and Media Library files from `ng serve` to the Go backend, so the browser sees
// one origin and the API needs no CORS. Point it elsewhere with API_URL, e.g.
// API_URL=http://localhost:18080 npm start.
const backend = {
  target: process.env.API_URL || 'http://localhost:8080',
  secure: false,
  changeOrigin: true,
};

module.exports = {
  '/v1': backend,
  '/uploads': backend,
};
