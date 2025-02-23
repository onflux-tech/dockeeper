import Docker from "dockerode";
import { config } from "../config/environment";

interface VolumeInfo {
  Name: string;
  CreatedAt: string;
  Labels: { [key: string]: string };
}

export async function cleanUnusedVolumes(docker: Docker): Promise<void> {
  if (!config.volumes.enabled) {
    console.log("Volume cleanup is disabled");
    return;
  }

  try {
    const volumes = await docker.listVolumes();
    const containers = await docker.listContainers({ all: true });
    const usedVolumes = new Set<string>();

    containers.forEach((container) => {
      if (container.Mounts) {
        container.Mounts.forEach((mount) => {
          if (mount.Name) {
            usedVolumes.add(mount.Name);
          }
        });
      }
    });

    const retentionDate = new Date();
    retentionDate.setDate(
      retentionDate.getDate() - config.volumes.retentionDays
    );

    const unusedVolumes = (volumes.Volumes as unknown as VolumeInfo[]).filter(
      (volume: VolumeInfo) => {
        if (usedVolumes.has(volume.Name)) {
          return false;
        }

        const createdAt = new Date(volume.CreatedAt);
        return createdAt < retentionDate;
      }
    );

    console.log(
      `Found ${unusedVolumes.length} unused volumes older than ${config.volumes.retentionDays} days`
    );

    for (const volume of unusedVolumes) {
      try {
        await docker.getVolume(volume.Name).remove();
        console.log(
          `Volume removed: ${volume.Name} (created at ${volume.CreatedAt})`
        );
      } catch (err) {
        console.warn(`Could not remove volume ${volume.Name}:`, err);
      }
    }
  } catch (error) {
    console.error("Error cleaning volumes:", error);
  }
}
