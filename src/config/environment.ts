import "dotenv/config";

export type NotificationService = "evolution" | "wuzapi";

export interface MonitoringConfig {
  mode: "all" | "selective";
  containers: string[];
}

export const config = {
  cleanupInterval: Number(process.env.CLEANUP_INTERVAL_HOURS || 24),
  docker: {
    socketPath:
      process.platform === "win32"
        ? "//./pipe/docker_engine"
        : "/var/run/docker.sock",
    isSwarm: process.env.DOCKER_MODE === "swarm",
  },
  notification: {
    service: (process.env.NOTIFICATION_SERVICE || "") as NotificationService,
    api: process.env.NOTIFICATION_URL || "",
    key: process.env.NOTIFICATION_KEY || "",
    number: process.env.NOTIFICATION_NUMBER || "",
  },
  monitoring: {
    healthChecks: parseHealthChecks(process.env.HEALTH_CHECKS),
    throttleInterval: 60 * 1000,
  },
  cleanup: {
    retentionDays: Number(process.env.CLEANUP_RETENTION_DAYS || 0),
    enabled: Boolean(Number(process.env.CLEANUP_RETENTION_DAYS || 0) > 0),
  },
};

function parseHealthChecks(envValue?: string): MonitoringConfig {
  if (!envValue || envValue.trim() === "") {
    return {
      mode: "all",
      containers: [],
    };
  }

  return {
    mode: "selective",
    containers: envValue.split(";").filter(Boolean),
  };
}
