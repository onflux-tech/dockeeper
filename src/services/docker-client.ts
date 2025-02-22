import Docker from "dockerode";
import { config } from "../config/environment";

export function createDockerClient(): Docker {
  return new Docker({ socketPath: config.docker.socketPath });
}
