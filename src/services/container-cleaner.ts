import Docker from "dockerode";

export async function cleanExitedContainers(docker: Docker): Promise<void> {
  try {
    const containers = await docker.listContainers({
      all: true,
      filters: { status: ["exited"] },
    });
    console.log(`Found ${containers.length} stopped containers.`);

    for (const containerInfo of containers) {
      const container = docker.getContainer(containerInfo.Id);
      await container.remove();
      console.log(`Container removed: ${containerInfo.Id}`);
    }
  } catch (error) {
    console.error("Error cleaning stopped containers:", error);
  }
}
