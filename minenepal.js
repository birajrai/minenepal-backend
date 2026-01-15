// minenepal.js
const express = require("express");
const cors = require("cors");
const fs = require("fs");
const path = require("path");
const { status } = require("minecraft-server-util");

const app = express();
const PORT = 3000;

app.use(cors());
app.use(express.json());

// Cache directories & TTL
const CACHE_DIR = path.join(__dirname, "cache");
const CACHE_TTL = 10 * 1000; // 10 seconds

if (!fs.existsSync(CACHE_DIR)) fs.mkdirSync(CACHE_DIR);

// In-memory cache
const memoryCache = new Map();

// Utility: sanitize filename
function sanitizeFilename(ip, port) {
  return ip.replace(/[:/\\]/g, "_") + `_${port}.json`;
}

// Main handler
async function checkServer(req, res) {
  const ip = req.params.ip;
  const port = parseInt(req.params.port) || 25565;
  const cacheKey = `${ip}:${port}`;
  const filePath = path.join(CACHE_DIR, sanitizeFilename(ip, port));

  // 1️⃣ Check memory cache
  if (memoryCache.has(cacheKey)) {
    const cached = memoryCache.get(cacheKey);
    if (Date.now() - cached.timestamp < CACHE_TTL) {
      return res.json(cached.data);
    }
  }

  // 2️⃣ Check file cache
  if (fs.existsSync(filePath)) {
    try {
      const fileData = JSON.parse(fs.readFileSync(filePath, "utf-8"));
      if (Date.now() - fileData.timestamp < CACHE_TTL) {
        memoryCache.set(cacheKey, fileData); // load into memory
        return res.json(fileData.data);
      }
    } catch (err) {
      console.error("Error reading cache file:", filePath, err);
    }
  }

  // 3️⃣ Ping server
  try {
    const result = await status(ip, port, { timeout: 5000, enableSRV: true });
    const data = {
      online: true,
      ip,
      port,
      ping: result.roundTripLatency,
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

    const cacheEntry = { timestamp: Date.now(), data };
    memoryCache.set(cacheKey, cacheEntry); // memory cache
    fs.writeFileSync(filePath, JSON.stringify(cacheEntry, null, 2)); // file cache

    res.json(data);
  } catch (err) {
    const offlineData = { online: false, ip, port, ping: null, error: "Server offline or unreachable" };
    const cacheEntry = { timestamp: Date.now(), data: offlineData };

    memoryCache.set(cacheKey, cacheEntry);
    fs.writeFileSync(filePath, JSON.stringify(cacheEntry, null, 2));

    res.status(404).json(offlineData);
  }
}

// Routes
app.get("/api/server/status/:ip/:port", checkServer);
app.get("/api/server/status/:ip", checkServer);

app.listen(PORT, () => {
  console.log(`🚀 MineNepal Server Status API running on http://localhost:${PORT}`);
});
