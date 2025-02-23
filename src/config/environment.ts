import "dotenv/config";

export const config = {
  cleanupInterval: Number(process.env.CLEANUP_INTERVAL_HOURS || 24),
  docker: {
    socketPath:
      process.platform === "win32"
        ? "//./pipe/docker_engine"
        : "/var/run/docker.sock",
    isSwarm: process.env.DOCKER_MODE === "swarm",
  },
  notificationService: process.env.NOTIFICATION_SERVICE || "evolution",
  evolution: {
    instance: process.env.EVO_INSTANCE || "",
    apiKey: process.env.EVO_APIKEY || "",
    number: process.env.EVO_NUMBER || "",
  },
  meow: {
    instance: process.env.MEOW_INSTANCE || "",
    apiKey: process.env.MEOW_APIKEY || "",
    number: process.env.MEOW_NUMBER || "",
  },
  monitoring: {
    healthChecks: process.env.HEALTH_CHECKS
      ? process.env.HEALTH_CHECKS.split(";").filter(Boolean)
      : [],
    throttleInterval: 60 * 1000,
  },
};
