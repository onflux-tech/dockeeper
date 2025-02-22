import Docker from "dockerode";

export async function cleanBuildCache(docker: Docker): Promise<void> {
  try {
    const result = await new Promise<{ SpaceReclaimed: number }>(
      (resolve, reject) => {
        docker.modem.dial(
          {
            method: "POST",
            path: "/build/prune",
            statusCodes: {
              200: true,
            },
          },
          (err, data) => {
            if (err) return reject(err);
            if (
              !data ||
              typeof data !== "object" ||
              !("SpaceReclaimed" in data)
            ) {
              return reject(new Error("Unexpected data in Docker response"));
            }
            resolve(data as { SpaceReclaimed: number });
          }
        );
      }
    );

    const spaceCleaned = (result.SpaceReclaimed / 1024 / 1024).toFixed(2);
    console.log(`Build cache cleaned: ${spaceCleaned}MB reclaimed`);
  } catch (error) {
    console.error("Error cleaning build cache:", error);
  }
}
