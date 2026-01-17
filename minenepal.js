// minenepal.js
"use strict";

const express = require("express");
const cors = require("cors");
const path = require("path");
const dns = require("dns").promises;
const fs = require("fs/promises");
const { status } = require("minecraft-server-util");
const sharp = require("sharp");
const os = require("os");

const app = express();
const PORT = 10000;

// ======================
// Health Metrics
// ======================
let requestCount = 0;
let responseTimes = [];
const MAX_RESPONSE_TIMES = 100;
let healthCache = null;
const HEALTH_TTL = 5000; // 5 seconds

// ======================
// Middleware
// ======================
app.use(cors());
app.use(express.json());
app.disable("x-powered-by");
app.set("trust proxy", true);

// Metrics middleware
app.use((req, res, next) => {
  const start = Date.now();
  res.on('finish', () => {
    const duration = Date.now() - start;
    requestCount++;
    responseTimes.push(duration);
    if (responseTimes.length > MAX_RESPONSE_TIMES) {
      responseTimes.shift();
    }
  });
  next();
});

// ======================
// Paths & Cache Config
// ======================
const CACHE_DIR = path.join(__dirname, "cache");
const ICON_DIR = path.join(CACHE_DIR, "icons");
const TTL = 15 * 1000; // 15 seconds

// Serve icons
app.use("/icons", express.static(ICON_DIR));

// ======================
// Memory Cache
// ======================
const memoryCache = new Map();

// ======================
// Auto-refresh servers
// ======================
const autoRefreshServers = [
  "play.hypixel.net",
  "play.craftnepal.com",
  "mcnpnetwork.com"
];

// ======================
// Init directories
// ======================
(async () => {
  await fs.mkdir(ICON_DIR, { recursive: true });
})();

// ======================
// Cache cleanup
// ======================
setInterval(() => {
  const now = Date.now();
  for (const [key, value] of memoryCache) {
    if (now - value.timestamp > TTL) {
      memoryCache.delete(key);
    }
  }
}, TTL);

// ======================
// Utils
// ======================
const sanitize = (s) => s.replace(/[:/\\]/g, "_");

async function getSystemMetrics() {
  const memUsage = process.memoryUsage();
  const cpus = os.cpus();
  const loadAvg = os.loadavg();

  // Calculate CPU usage (simplified)
  const totalIdle = cpus.reduce((sum, cpu) => sum + cpu.times.idle, 0);
  const totalTick = cpus.reduce((sum, cpu) => sum + Object.values(cpu.times).reduce((a, b) => a + b, 0), 0);
  const cpuUsage = ((totalTick - totalIdle) / totalTick) * 100;

  // Memory usage
  const memoryUsage = {
    rss: (memUsage.rss / 1024 / 1024).toFixed(2) + ' MB',
    heapUsed: (memUsage.heapUsed / 1024 / 1024).toFixed(2) + ' MB',
    heapTotal: (memUsage.heapTotal / 1024 / 1024).toFixed(2) + ' MB',
    external: (memUsage.external / 1024 / 1024).toFixed(2) + ' MB'
  };

  // Storage usage (cache directory size)
  let storageUsage = '0 MB';
  try {
    const cacheSize = await getDirectorySize(CACHE_DIR);
    storageUsage = (cacheSize / 1024 / 1024).toFixed(2) + ' MB';
  } catch {}

  return {
    memory: memoryUsage,
    cpu: {
      usage: cpuUsage.toFixed(2) + '%',
      loadAverage: loadAvg.map(l => l.toFixed(2))
    },
    storage: storageUsage
  };
}

async function getDirectorySize(dirPath) {
  let totalSize = 0;
  const files = await fs.readdir(dirPath, { withFileTypes: true });
  for (const file of files) {
    const filePath = path.join(dirPath, file.name);
    if (file.isDirectory()) {
      totalSize += await getDirectorySize(filePath);
    } else {
      const stat = await fs.stat(filePath);
      totalSize += stat.size;
    }
  }
  return totalSize;
}

const jsonFile = (ip, port) =>
  path.join(CACHE_DIR, `${sanitize(ip)}_${port}.json`);

const iconFile = (ip, port) =>
  path.join(ICON_DIR, `${sanitize(ip)}_${port}.png`);

async function saveIcon(base64, ip, port) {
  try {
    const data = base64.replace(/^data:image\/png;base64,/, "");
    const buffer = Buffer.from(data, "base64");

    const resized = await sharp(buffer)
      .resize(32, 32)
      .png()
      .toBuffer();

    const filePath = iconFile(ip, port);
    await fs.writeFile(filePath, resized);

    return `/icons/${path.basename(filePath)}`;
  } catch {
    return null;
  }
}

// ======================
// Core status function
// ======================
async function getServerStatus(ip, port = 25565) {
  const key = `${ip}:${port}`;
  const filePath = jsonFile(ip, port);

  // Memory cache
  const mem = memoryCache.get(key);
  if (mem && Date.now() - mem.timestamp < TTL) return mem.data;

  // File cache
  try {
    const file = JSON.parse(await fs.readFile(filePath, "utf8"));
    if (Date.now() - file.timestamp < TTL) {
      memoryCache.set(key, file);
      return file.data;
    }
  } catch {}

  // Ping server
  try {
    const res = await status(ip, port, {
      timeout: 5000,
      enableSRV: true
    });

    let raw_ip = null;
    try {
      const lookup = await dns.lookup(res.host || ip);
      raw_ip = lookup.address;
    } catch {}

    let icon = null;
    if (res.favicon) {
      icon = await saveIcon(res.favicon, ip, res.port || port);
    }

    const data = {
      online: true,
      ip,
      host: res.host || ip,
      raw_ip,
      port: res.port || port,
      ping: res.roundTripLatency,
      version: res.version.name,
      players: {
        online: res.players.online,
        max: res.players.max
      },
      motd: {
        clean: res.motd.clean,
        raw: res.motd.raw,
        html: res.motd.html
      },
      icon
    };

    const entry = { timestamp: Date.now(), data };
    memoryCache.set(key, entry);
    await fs.writeFile(filePath, JSON.stringify(entry, null, 2));

    return data;
  } catch {
    const offline = {
      online: false,
      ip,
      host: ip,
      raw_ip: null,
      port,
      ping: null,
      error: "Server offline or unreachable",
      icon: null
    };

    const entry = { timestamp: Date.now(), data: offline };
    memoryCache.set(key, entry);
    await fs.writeFile(filePath, JSON.stringify(entry, null, 2));

    return offline;
  }
}

// ======================
// Auto refresh loop
// ======================
async function refreshAllServers() {
  for (let s of autoRefreshServers) {
    let ip = s;
    let port = 25565;

    if (s.includes(":")) {
      const p = s.split(":");
      ip = p[0];
      port = parseInt(p[1]) || 25565;
    }

    try {
      await getServerStatus(ip, port);
    } catch {}
  }
}

setInterval(refreshAllServers, TTL);
refreshAllServers();

// ======================
// Routes
// ======================

// Health endpoint
app.get("/", async (req, res) => {
  const now = Date.now();
  if (healthCache && now - healthCache.timestamp < HEALTH_TTL) {
    return res.json(healthCache.data);
  }

  const metrics = await getSystemMetrics();
  const avgResponseTime = responseTimes.length > 0
    ? (responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length).toFixed(2) + ' ms'
    : '0 ms';

  const health = {
    status: "healthy",
    uptime: process.uptime().toFixed(2) + ' seconds',
    responseTime: avgResponseTime,
    requests: requestCount,
    ...metrics
  };

  healthCache = { timestamp: now, data: health };
  res.json(health);
});

// Bulk
app.get("/api/server/status/bulk", async (req, res) => {
  const list = req.query.servers;
  if (!list) return res.status(400).json({ error: "No servers provided" });

  const servers = list.split(",").map(s => s.trim()).filter(Boolean);
  const results = {};

  await Promise.all(
    servers.map(async (s) => {
      let ip = s;
      let port = 25565;

      if (s.includes(":")) {
        const p = s.split(":");
        ip = p[0];
        port = parseInt(p[1]) || 25565;
      }

      results[`${ip}:${port}`] = await getServerStatus(ip, port);
    })
  );

  res.json(results);
});

// Single (with port)
app.get("/api/server/status/:ip/:port", async (req, res) => {
  const data = await getServerStatus(
    req.params.ip,
    parseInt(req.params.port)
  );
  res.status(data.online ? 200 : 404).json(data);
});

// Single (default port)
app.get("/api/server/status/:ip", async (req, res) => {
  const data = await getServerStatus(req.params.ip);
  res.status(data.online ? 200 : 404).json(data);
});

// ======================
// Start server
// ======================
app.listen(PORT, () => {
  console.log(`🚀 MineNepal API running on http://localhost:${PORT}`);
});
