// minenepal.js
const express = require("express");
const cors = require("cors");
const fs = require("fs");
const path = require("path");
const dns = require("dns").promises;
const { status } = require("minecraft-server-util");

const app = express();
const PORT = 3000;

app.use(cors());
app.use(express.json());

// Cache config
const CACHE_DIR = path.join(__dirname, "cache");
const TTL = 15 * 1000; // 15 seconds

if (!fs.existsSync(CACHE_DIR)) fs.mkdirSync(CACHE_DIR);

// Memory cache
const memoryCache = new Map();

// Utility to sanitize filenames for JSON cache
function sanitizeFilename(ip, port) {
  return ip.replace(/[:/\\]/g, "_") + `_${port}.json`;
}

// Function to get server status with memory + JSON cache
async function getServerStatus(ip, port = 25565) {
  const cacheKey = `${ip}:${port}`;
  const filePath = path.join(CACHE_DIR, sanitizeFilename(ip, port));

  // 1️⃣ Check memory cache
  if (memoryCache.has(cacheKey)) {
    const cached = memoryCache.get(cacheKey);
    if (Date.now() - cached.timestamp < TTL) return cached.data;
  }

  // 2️⃣ Check file cache
  if (fs.existsSync(filePath)) {
    try {
      const fileData = JSON.parse(fs.readFileSync(filePath, "utf-8"));
      if (Date.now() - fileData.timestamp < TTL) {
        memoryCache.set(cacheKey, fileData);
        return fileData.data;
      }
    } catch (err) {
      console.error("Error reading cache file:", filePath, err);
    }
  }

  // 3️⃣ Ping server
  try {
    const result = await status(ip, port, { timeout: 5000, enableSRV: true });

    // resolve numeric IP safely
    let raw_ip = null;
    if (result.host) {
      try {
        const dnsResult = await dns.lookup(result.host);
        raw_ip = dnsResult.address;
      } catch {
        raw_ip = null;
      }
    }

    const data = {
      online: true,
      ip,               // requested domain
      host: result.host || ip, // SRV-resolved host or fallback
      raw_ip,           // numeric IP or null
      port: result.port || port, // actual port pinged or fallback
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
      icon: result.favicon || null,
    };

    const cacheEntry = { timestamp: Date.now(), data };
    memoryCache.set(cacheKey, cacheEntry);
    fs.writeFileSync(filePath, JSON.stringify(cacheEntry, null, 2));

    return data;
  } catch (err) {
    // Offline or unreachable
    const offlineData = {
      online: false,
      ip,
      host: ip,    // fallback
      raw_ip: null,
      port,        // port we tried
      ping: null,
      error: "Server offline or unreachable",
      icon: null,
    };

    const cacheEntry = { timestamp: Date.now(), data: offlineData };
    memoryCache.set(cacheKey, cacheEntry);
    fs.writeFileSync(filePath, JSON.stringify(cacheEntry, null, 2));

    return offlineData;
  }
}

/**
 * 1️⃣ Bulk route MUST be defined BEFORE dynamic routes
 */
app.get("/api/server/status/bulk", async (req, res) => {
  const serversParam = req.query.servers;

  if (!serversParam) return res.status(400).json({ error: "No servers provided" });

  const servers = serversParam.toString().split(",").map(s => s.trim()).filter(s => s);

  if (servers.length === 0) return res.status(400).json({ error: "No valid servers provided" });

  const results = {};

  await Promise.all(
    servers.map(async (s) => {
      let ip = s;
      let port = 25565;

      if (ip.includes(":")) {
        const parts = ip.split(":");
        ip = parts[0];
        port = parseInt(parts[1]) || 25565;
      }

      results[`${ip}:${port}`] = await getServerStatus(ip, port);
    })
  );

  res.json(results);
});

// 2️⃣ Single server routes
app.get("/api/server/status/:ip/:port", async (req, res) => {
  const ip = req.params.ip;
  const port = parseInt(req.params.port);
  const data = await getServerStatus(ip, port);
  res.status(data.online ? 200 : 404).json(data);
});

app.get("/api/server/status/:ip", async (req, res) => {
  const ip = req.params.ip;
  const data = await getServerStatus(ip);
  res.status(data.online ? 200 : 404).json(data);
});

// Start server
app.listen(PORT, () => {
  console.log(`🚀 MineNepal Server Status API running on http://localhost:${PORT}`);
});
