import Docker from "dockerode";
import { config } from "../config/environment";

export async function cleanUnusedImages(docker: Docker): Promise<void> {
  try {
    const retentionMs = config.cleanup.retentionDays * 24 * 60 * 60 * 1000;
    const cutoffDate = new Date(Date.now() - retentionMs);

    const allImages = await docker.listImages();
    console.log(`Total images found: ${allImages.length}`);

    const containers = await docker.listContainers({ all: true });
    const usedImageIds = new Set(
      containers.map((container) => container.ImageID)
    );

    const unusedImages = allImages.filter((image) => {
      const created = new Date(image.Created * 1000);
      const isUnused = !usedImageIds.has(image.Id);
      const isOld = config.cleanup.enabled ? created < cutoffDate : true;
      const isNotNone = !image.RepoTags?.some((tag) => tag.includes("<none>"));

      return isUnused && isOld && isNotNone;
    });

    console.log(
      `Found ${unusedImages.length} unused images older than ${config.cleanup.retentionDays} days`
    );

    for (const imageInfo of unusedImages) {
      const image = docker.getImage(imageInfo.Id);
      const tags = imageInfo.RepoTags?.join(", ") || "no tags";
      try {
        await image.remove();
        console.log(`Image removed: ${tags} (${imageInfo.Id})`);
      } catch (err) {
        console.warn(`Could not remove image ${tags}: ${err}`);
      }
    }
  } catch (error) {
    console.error("Error cleaning images:", error);
  }
}
