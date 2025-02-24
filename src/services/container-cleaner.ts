import Docker from "dockerode";
import { config } from "../config/environment";

export async function cleanExitedContainers(docker: Docker): Promise<void> {
  try {
    const retentionMs = config.cleanup.retentionDays * 24 * 60 * 60 * 1000;
    const cutoffDate = new Date(Date.now() - retentionMs);

    const containers = await docker.listContainers({
      all: true,
      filters: { status: ["exited"] },
    });

    const containersToRemove = config.cleanup.enabled
      ? containers.filter((container) => {
          const stoppedAt = new Date(container.Status);
          return stoppedAt < cutoffDate;
        })
      : containers;

    console.log(
      `Found ${containersToRemove.length} stopped containers older than ${config.cleanup.retentionDays} days`
    );

    for (const containerInfo of containersToRemove) {
      const container = docker.getContainer(containerInfo.Id);
      await container.remove();
      console.log(
        `Container removed: ${containerInfo.Id} (${containerInfo.Names.join(
          ", "
        )})`
      );
    }
  } catch (error) {
    console.error("Error cleaning stopped containers:", error);
  }
}
