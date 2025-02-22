import Docker from "dockerode";

export async function cleanUnusedImages(docker: Docker): Promise<void> {
  try {
    const allImages = await docker.listImages();
    console.log(`Total images found: ${allImages.length}`);

    const containers = await docker.listContainers({ all: true });
    const usedImageIds = new Set(
      containers.map((container) => container.ImageID)
    );

    const unusedImages = allImages.filter(
      (image) =>
        !usedImageIds.has(image.Id) &&
        !image.RepoTags?.some((tag) => tag.includes("<none>"))
    );

    console.log(`Found ${unusedImages.length} unused images.`);

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
