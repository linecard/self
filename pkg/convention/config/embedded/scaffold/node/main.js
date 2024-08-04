const express = require('express');

const app = express();
const PORT = process.env.AWS_LWA_PORT || 8081;
const HOST = '0.0.0.0';

app.get('/', (req, res) => {
  res.json(req.headers);
});

app.use((req, res) => {
  res.status(404).json({ error: 'Not found' });
});

app.listen(PORT, HOST, () => {
  console.log(`Server is running on http://${HOST}:${PORT}`);
});
