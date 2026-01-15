const express = require("express");
const cors = require("cors");
const { status } = require("minecraft-server-util");

const app = express();
const PORT = 3000;

app.use(cors());

async function handler(req, res) {
  const ip = req.params.ip;
  const port = parseInt(req.params.port) || 25565;

  try {
    const result = await status(ip, port, {
      timeout: 5000,
      enableSRV: true,
    });

    res.json({
      online: true,
      ip,
      port,
      version: result.version.name,
      players: {
        online: result.players.online,
        max: result.players.max,
      },
      motd: result.motd.clean,
      latency: result.roundTripLatency,
    });
  } catch (err) {
    res.status(404).json({
      online: false,
      ip,
      port,
      error: "Server is offline or unreachable",
    });
  }
}

// IMPORTANT: order matters
app.get("/api/server/status/:ip/:port", handler);
app.get("/api/server/status/:ip", handler);

app.listen(PORT, () => {
  console.log(`🚀 API running on http://localhost:${PORT}`);
});
