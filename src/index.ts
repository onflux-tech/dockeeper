import "dotenv/config";
import { config } from "./config/environment";
import { createDockerClient } from "./services/docker-client";
import { cleanExitedContainers } from "./services/container-cleaner";
import { cleanUnusedImages } from "./services/image-cleaner";
import { cleanBuildCache } from "./services/cache-cleaner";
import { DockerMonitor } from "./services/docker-monitor";
import { HttpServer } from "./services/http-server";

async function cleanup(): Promise<void> {
  const docker = createDockerClient();
  console.log("Starting Docker cleanup...");

  await cleanExitedContainers(docker);
  await cleanUnusedImages(docker);
  await cleanBuildCache(docker);

  console.log("Docker cleanup completed.");
}

async function startPeriodicCleanup(): Promise<void> {
  const docker = createDockerClient();

  const httpServer = new HttpServer(docker);
  httpServer.start();

  const monitor = new DockerMonitor(docker);
  await monitor.startMonitoring();

  while (true) {
    await cleanup();
    const intervalHours = config.cleanupInterval;
    console.log(`Waiting ${intervalHours} hours until next execution...`);
    await new Promise((resolve) =>
      setTimeout(resolve, intervalHours * 60 * 60 * 1000)
    );
  }
}

startPeriodicCleanup().catch((err) =>
  console.error("Error in main execution:", err)
);
