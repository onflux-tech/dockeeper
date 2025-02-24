import { WebSocketServer, WebSocket } from "ws";
import { Server } from "http";
import { Server as HttpsServer } from "https";
import { databaseService } from "./database";

export class WebSocketService {
  private wss: WebSocketServer;
  private clients: Set<WebSocket> = new Set();
  private broadcastThrottle: NodeJS.Timeout | null = null;
  private queuedMetrics: any = null;

  constructor(server: Server | HttpsServer) {
    this.wss = new WebSocketServer({
      server,
      path: "/",
      perMessageDeflate: {
        zlibDeflateOptions: {
          chunkSize: 1024,
          memLevel: 7,
          level: 3,
        },
        zlibInflateOptions: {
          chunkSize: 10 * 1024,
        },
        clientNoContextTakeover: true,
        serverNoContextTakeover: true,
        serverMaxWindowBits: 10,
        concurrencyLimit: 10,
        threshold: 1024,
      },
    });

    this.setupWebSocket();
  }

  private setupWebSocket(): void {
    this.wss.on("connection", (ws: WebSocket) => {
      this.clients.add(ws);

      ws.on("close", () => {
        this.clients.delete(ws);
      });

      this.sendMetrics(ws);
    });
  }

  private sendMetrics(ws: WebSocket): void {
    const metrics = databaseService.getMetrics();
    ws.send(JSON.stringify({ type: "metrics", data: metrics }));
  }

  public broadcastMetrics(): void {
    const metrics = databaseService.getMetrics();
    this.queuedMetrics = metrics;

    if (!this.broadcastThrottle) {
      this.broadcastThrottle = setTimeout(() => {
        if (this.queuedMetrics) {
          const message = JSON.stringify({
            type: "metrics",
            data: this.queuedMetrics,
          });

          this.clients.forEach((client) => {
            if (client.readyState === WebSocket.OPEN) {
              client.send(message);
            }
          });
        }
        this.broadcastThrottle = null;
        this.queuedMetrics = null;
      }, 2000);
    }
  }
}
