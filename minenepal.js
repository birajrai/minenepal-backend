// minenepal.js
const express = require("express");
const cors = require("cors");
const { status } = require("minecraft-server-util");

const app = express();
const PORT = 3000;

app.use(cors());
app.use(express.json());

// In-memory cache to reduce ping spam
const cache = new Map();
const CACHE_TTL = 10 * 1000; // 10 seconds

// Handler function
async function checkServer(req, res) {
  const ip = req.params.ip;
  const port = parseInt(req.params.port) || 25565;
  const cacheKey = `${ip}:${port}`;

  // Serve cached response if exists
  if (cache.has(cacheKey)) {
    return res.json(cache.get(cacheKey));
  }

  try {
    const result = await status(ip, port, {
      timeout: 5000,
      enableSRV: true, // resolves domain SRV records
    });

    const response = {
      online: true,
      ip,
      port,
      ping: result.roundTripLatency, // ping in ms
      version: result.version.name,
      players: {
        online: result.players.online,
        max: result.players.max,
      },
      motd: {
        clean: result.motd.clean,
        raw: result.motd.raw,
        html: result.motd.html,
      },
    };

    // Cache the response
    cache.set(cacheKey, response);
    setTimeout(() => cache.delete(cacheKey), CACHE_TTL);

    res.json(response);
  } catch (err) {
    const offlineResponse = {
      online: false,
      ip,
      port,
      ping: null,
      error: "Server is offline or unreachable",
    };

    // Cache offline response too
    cache.set(cacheKey, offlineResponse);
    setTimeout(() => cache.delete(cacheKey), CACHE_TTL);

    res.status(404).json(offlineResponse);
  }
}

// Routes
app.get("/api/server/status/:ip/:port", checkServer);
app.get("/api/server/status/:ip", checkServer);

app.listen(PORT, () => {
  console.log(`🚀 MineNepal Server Status API running on http://localhost:${PORT}`);
});
