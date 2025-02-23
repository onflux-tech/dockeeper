import "dotenv/config";

export type NotificationService = "evolution" | "wuzapi";

export const config = {
  cleanupInterval: Number(process.env.CLEANUP_INTERVAL_HOURS || 24),
  docker: {
    socketPath:
      process.platform === "win32"
        ? "//./pipe/docker_engine"
        : "/var/run/docker.sock",
    isSwarm: process.env.DOCKER_MODE === "swarm",
    nodeId: process.env.NODE_ID || "standalone",
  },
  notification: {
    service: (process.env.NOTIFICATION_SERVICE || "") as NotificationService,
    api: process.env.NOTIFICATION_URL || "",
    key: process.env.NOTIFICATION_KEY || "",
    number: process.env.NOTIFICATION_NUMBER || "",
  },
  monitoring: {
    healthChecks: process.env.HEALTH_CHECKS
      ? process.env.HEALTH_CHECKS.split(";").filter(Boolean)
      : [],
    throttleInterval: 60 * 1000,
  },
  volumes: {
    retentionDays: Number(process.env.VOLUME_RETENTION_DAYS || 0),
    enabled: Boolean(Number(process.env.VOLUME_RETENTION_DAYS || 0) > 0),
  },
};
