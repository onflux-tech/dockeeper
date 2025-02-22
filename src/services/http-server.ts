import express from "express";
import Docker from "dockerode";
import { cleanExitedContainers } from "./container-cleaner";
import { cleanUnusedImages } from "./image-cleaner";
import { cleanBuildCache } from "./cache-cleaner";

interface StatusResponse {
  status: "ok" | "error";
  lastRun: Date | null;
  mode: string;
  running: boolean;
}

export class HttpServer {
  private app = express();
  private server: any;
  private lastRun: Date | null = null;
  private isRunning = false;

  constructor(private docker: Docker) {
    this.setupRoutes();
  }

  private setupRoutes(): void {
    this.app.get("/health", (_, res) => {
      const status: StatusResponse = {
        status: "ok",
        lastRun: this.lastRun,
        mode: process.env.DOCKER_MODE || "standalone",
        running: this.isRunning,
      };
      res.json(status);
    });

    this.app.post("/run", async (_, res) => {
      if (this.isRunning) {
        return res.status(409).json({
          message: "Maintenance is already running",
        });
      }

      this.runMaintenance();
      res.json({
        message: "Maintenance started in background",
      });
    });
  }

  private async runMaintenance(): Promise<void> {
    if (this.isRunning) return;

    this.isRunning = true;
    try {
      console.log("Starting Docker cleanup via API...");
      await cleanExitedContainers(this.docker);
      await cleanUnusedImages(this.docker);
      await cleanBuildCache(this.docker);
      this.lastRun = new Date();
      console.log("Docker cleanup via API completed.");
    } catch (error) {
      console.error("Error during maintenance:", error);
    } finally {
      this.isRunning = false;
    }
  }

  public start(port: number = 8080): void {
    this.server = this.app.listen(port, () => {
      console.log(`HTTP server started on port ${port}`);
    });
  }

  public stop(): void {
    if (this.server) {
      this.server.close();
    }
  }
}
