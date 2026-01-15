// minenepal.js
const express = require("express");
const cors = require("cors");
const fs = require("fs");
const path = require("path");
const dns = require("dns").promises;
const { status } = require("minecraft-server-util");
const sharp = require("sharp");

const app = express();
const PORT = 3000;

app.use(cors());
app.use(express.json());

// Serve icons statically
app.use("/icons", express.static(path.join(__dirname, "cache/icons")));

// Cache config
const CACHE_DIR = path.join(__dirname, "cache");
const ICON_DIR = path.join(CACHE_DIR, "icons");
const TTL = 15 * 1000; // 15 seconds

if (!fs.existsSync(CACHE_DIR)) fs.mkdirSync(CACHE_DIR);
if (!fs.existsSync(ICON_DIR)) fs.mkdirSync(ICON_DIR);

// Memory cache
const memoryCache = new Map();

// List of servers to auto-refresh
const autoRefreshServers = [
  "play.hypixel.net",
  "play.craftnepal.com",
  "mcnpnetwork.com",
  // Add more servers here
];

// Utility to sanitize filenames for JSON cache
function sanitizeFilename(ip, port) {
  return ip.replace(/[:/\\]/g, "_") + `_${port}.json`;
}

// Utility to sanitize icon filenames
function sanitizeIconName(ip, port) {
  return ip.replace(/[:/\\]/g, "_") + `_${port}.png`;
}

// Function to save icon to file and return URL
async function saveIcon(iconBase64, ip, port) {
  try {
    const base64Data = iconBase64.replace(/^data:image\/png;base64,/, "");
    const buffer = Buffer.from(base64Data, "base64");

    // Resize to 32x32 to reduce size
    const resizedBuffer = await sharp(buffer).resize(32, 32).png().toBuffer();

    const iconName = sanitizeIconName(ip, port);
    const iconPath = path.join(ICON_DIR, iconName);

    fs.writeFileSync(iconPath, resizedBuffer);

    // Return URL for API
    return `/icons/${iconName}`;
  } catch (err) {
    console.error("Failed to save icon for", ip, port, err);
    return null;
  }
}

// Function to get server status
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

    // Resolve numeric IP
    let raw_ip = null;
    if (result.host) {
      try {
        const dnsResult = await dns.lookup(result.host);
        raw_ip = dnsResult.address;
      } catch {}
    }

    // Handle icon
    let iconUrl = null;
    if (result.favicon) {
      iconUrl = await saveIcon(result.favicon, ip, result.port || port);
    }

    const data = {
      online: true,
      ip,                   // requested domain
      host: result.host || ip, // SRV-resolved host
      raw_ip,               // numeric IP
      port: result.port || port, // actual port pinged
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
      icon: iconUrl,        // icon URL
    };

    const cacheEntry = { timestamp: Date.now(), data };
    memoryCache.set(cacheKey, cacheEntry);
    fs.writeFileSync(filePath, JSON.stringify(cacheEntry, null, 2));

    return data;
  } catch {
    // Offline or unreachable
    const offlineData = {
      online: false,
      ip,
      host: ip,
      raw_ip: null,
      port,
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

// Auto-refresh all servers every 15s
async function refreshAllServers() {
  for (const s of autoRefreshServers) {
    let ip = s;
    let port = 25565;

    if (ip.includes(":")) {
      const parts = ip.split(":");
      ip = parts[0];
      port = parseInt(parts[1]) || 25565;
    }

    try {
      await getServerStatus(ip, port);
    } catch {}
  }
}

// Start background refresh loop
setInterval(refreshAllServers, TTL);

// Initial refresh
refreshAllServers();

// Bulk endpoint (cache-aware)
app.get("/api/server/status/bulk", async (req, res) => {
  const serversParam = req.query.servers;

  if (!serversParam) return res.status(400).json({ error: "No servers provided" });

  const servers = serversParam.toString().split(",").map(s => s.trim()).filter(s => s);

  if (servers.length === 0) return res.status(400).json({ error: "No valid servers provided" });

  const results = {};
  const toPing = [];

  for (let s of servers) {
    let ip = s;
    let port = 25565;

    if (ip.includes(":")) {
      const parts = ip.split(":");
      ip = parts[0];
      port = parseInt(parts[1]) || 25565;
    }

    const cacheKey = `${ip}:${port}`;
    const filePath = path.join(CACHE_DIR, sanitizeFilename(ip, port));

    let cachedData = null;

    // Memory cache
    if (memoryCache.has(cacheKey)) {
      const cached = memoryCache.get(cacheKey);
      if (Date.now() - cached.timestamp < TTL) cachedData = cached.data;
    }

    // File cache
    if (!cachedData && fs.existsSync(filePath)) {
      try {
        const fileData = JSON.parse(fs.readFileSync(filePath, "utf-8"));
        if (Date.now() - fileData.timestamp < TTL) {
          cachedData = fileData.data;
          memoryCache.set(cacheKey, fileData);
        }
      } catch {}
    }

    if (cachedData) {
      results[`${ip}:${port}`] = cachedData;
    } else {
      toPing.push({ ip, port, key: `${ip}:${port}` });
    }
  }

  // Ping only servers that need update (rare, thanks to auto-refresh)
  if (toPing.length > 0) {
    await Promise.all(
      toPing.map(async (server) => {
        results[server.key] = await getServerStatus(server.ip, server.port);
      })
    );
  }

  res.json(results);
});

// Single server routes
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
