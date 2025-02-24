import Docker from "dockerode";
import { databaseService } from "./database";
import { WebSocketService } from "./websocket-server";
import { config } from "../config/environment";

export class MetricsCollector {
  private docker: Docker;
  private wsService: WebSocketService;
  private collectInterval: NodeJS.Timeout | null = null;
  private previousStats: Map<string, any> = new Map();
  private metricsCache: Map<string, any> = new Map();
  private lastBroadcast: number = 0;
  private readonly BROADCAST_THROTTLE = 2000;
  private readonly BATCH_SIZE = 5;

  constructor(docker: Docker, wsService: WebSocketService) {
    this.docker = docker;
    this.wsService = wsService;
  }

  public async startCollecting(): Promise<void> {
    this.collectMetrics();
    this.collectInterval = setInterval(() => {
      this.collectMetrics();
    }, 1000);
  }

  public stopCollecting(): void {
    if (this.collectInterval) {
      clearInterval(this.collectInterval);
    }
  }

  private async collectMetrics(): Promise<void> {
    try {
      let containers;

      if (config.monitoring.healthChecks.mode === "all") {
        containers = await this.docker.listContainers({ all: true });
      } else {
        containers = await this.docker.listContainers({
          all: true,
          filters: {
            name: config.monitoring.healthChecks.containers,
          },
        });
      }

      if (config.monitoring.healthChecks.mode === "selective") {
        const monitoredNames = config.monitoring.healthChecks.containers.map(
          (name) => (name.startsWith("/") ? name.slice(1) : name)
        );
        databaseService.cleanUnmonitoredMetrics(monitoredNames);
      }

      for (let i = 0; i < containers.length; i += this.BATCH_SIZE) {
        const batch = containers.slice(i, i + this.BATCH_SIZE);
        await Promise.all(
          batch.map((container) => this.processContainer(container))
        );
      }

      const now = Date.now();
      if (now - this.lastBroadcast >= this.BROADCAST_THROTTLE) {
        this.wsService.broadcastMetrics();
        this.lastBroadcast = now;
      }
    } catch (error) {
      console.error("Error collecting metrics:", error);
    }
  }

  private async processContainer(
    container: Docker.ContainerInfo
  ): Promise<void> {
    const cachedMetrics = this.metricsCache.get(container.Id);
    const now = Date.now();

    if (cachedMetrics && now - cachedMetrics.timestamp < 5000) {
      return;
    }

    try {
      const stats = await this.docker
        .getContainer(container.Id)
        .stats({ stream: false });

      const metrics = await this.calculateContainerMetrics(container, stats);

      this.metricsCache.set(container.Id, {
        ...metrics,
        timestamp: now,
      });

      databaseService.upsertMetrics(metrics);
    } catch (error: any) {
      if (error.statusCode === 404) {
        console.warn(`Container ${container.Id} no longer exists. Skipping.`);
        this.metricsCache.delete(container.Id);
      }
    }
  }

  private async calculateContainerMetrics(
    container: Docker.ContainerInfo,
    stats: any
  ) {
    const previousStats = this.previousStats.get(container.Id);
    const timestamp = Date.now();
    const networks = stats.networks || {};
    const networkInterface = Object.keys(networks)[0] || null;

    let networkRxSpeed = 0;
    let networkTxSpeed = 0;

    if (networkInterface && previousStats) {
      const currentRx = networks[networkInterface].rx_bytes;
      const currentTx = networks[networkInterface].tx_bytes;
      const timeDiff = (timestamp - previousStats.timestamp) / 1000;

      networkRxSpeed = previousStats.rx_bytes
        ? (currentRx - previousStats.rx_bytes) / timeDiff
        : 0;

      networkTxSpeed = previousStats.tx_bytes
        ? (currentTx - previousStats.tx_bytes) / timeDiff
        : 0;

      this.previousStats.set(container.Id, {
        timestamp,
        rx_bytes: currentRx,
        tx_bytes: currentTx,
        read_bytes:
          stats.blkio_stats?.io_service_bytes_recursive?.[0]?.value || 0,
        write_bytes:
          stats.blkio_stats?.io_service_bytes_recursive?.[1]?.value || 0,
      });
    } else if (!previousStats) {
      this.previousStats.set(container.Id, {
        timestamp,
        rx_bytes: networkInterface
          ? networks[networkInterface]?.rx_bytes || 0
          : 0,
        tx_bytes: networkInterface
          ? networks[networkInterface]?.tx_bytes || 0
          : 0,
        read_bytes:
          stats.blkio_stats?.io_service_bytes_recursive?.[0]?.value || 0,
        write_bytes:
          stats.blkio_stats?.io_service_bytes_recursive?.[1]?.value || 0,
      });
    }

    const metrics = {
      id: container.Id,
      name: container.Names[0].replace(/^\//, ""),
      cpu_usage: this.calculateCpuUsage(stats),
      memory_usage: stats.memory_stats.usage || 0,
      memory_limit: stats.memory_stats.limit || 0,
      network_rx_speed: networkRxSpeed,
      network_tx_speed: networkTxSpeed,
      disk_read_speed: this.calculateDiskSpeed(
        stats,
        previousStats,
        "read",
        timestamp
      ),
      disk_write_speed: this.calculateDiskSpeed(
        stats,
        previousStats,
        "write",
        timestamp
      ),
      notifications_sent: await this.getNotificationCount(container.Id),
      status: container.State.toLowerCase(),
    };

    this.previousStats.set(container.Id, {
      timestamp,
      rx_bytes: networkInterface ? networks[networkInterface].rx_bytes : 0,
      tx_bytes: networkInterface ? networks[networkInterface].tx_bytes : 0,
      read_bytes:
        stats.blkio_stats?.io_service_bytes_recursive?.[0]?.value || 0,
      write_bytes:
        stats.blkio_stats?.io_service_bytes_recursive?.[1]?.value || 0,
    });

    return metrics;
  }

  private calculateDiskSpeed(
    stats: any,
    previousStats: any,
    type: "read" | "write",
    timestamp: number
  ): number {
    if (!previousStats) return 0;

    const index = type === "read" ? 0 : 1;
    const currentBytes =
      stats.blkio_stats?.io_service_bytes_recursive?.[index]?.value ?? 0;
    const previousBytes =
      type === "read" ? previousStats.read_bytes : previousStats.write_bytes;
    const timeDiff = (timestamp - previousStats.timestamp) / 1000;

    return timeDiff > 0 ? (currentBytes - previousBytes) / timeDiff : 0;
  }

  private calculateCpuUsage(stats: any): number {
    const cpuDelta =
      stats.cpu_stats.cpu_usage.total_usage -
      stats.precpu_stats.cpu_usage.total_usage;
    const systemDelta =
      stats.cpu_stats.system_cpu_usage - stats.precpu_stats.system_cpu_usage;
    return systemDelta ? cpuDelta / systemDelta : 0;
  }

  private async getNotificationCount(containerId: string): Promise<number> {
    return await databaseService.getNotificationCount(containerId);
  }
}
