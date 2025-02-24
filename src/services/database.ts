import Database from "better-sqlite3";
import path from "path";

interface ContainerMetrics {
  id: string;
  name: string;
  cpu_usage: number;
  memory_usage: number;
  memory_limit: number;
  network_rx_speed: number;
  network_tx_speed: number;
  disk_read_speed: number;
  disk_write_speed: number;
  notifications_sent: number;
  status: string;
  last_update: string;
}

export class DatabaseService {
  private db: Database.Database;

  constructor() {
    const dbPath = path.join(process.cwd(), "data", "metrics.db");
    this.db = new Database(dbPath);
    this.initializeDatabase();
  }

  private initializeDatabase(): void {
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS container_metrics (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL,
        cpu_usage REAL DEFAULT 0,
        memory_usage INTEGER DEFAULT 0,
        memory_limit INTEGER DEFAULT 0,
        network_rx_speed REAL DEFAULT 0,
        network_tx_speed REAL DEFAULT 0,
        disk_read_speed REAL DEFAULT 0,
        disk_write_speed REAL DEFAULT 0,
        notifications_sent INTEGER DEFAULT 0,
        status TEXT DEFAULT 'unknown',
        last_update DATETIME DEFAULT CURRENT_TIMESTAMP
      )
    `);
  }

  public upsertMetrics(metrics: Partial<ContainerMetrics>): void {
    const stmt = this.db.prepare(`
      INSERT INTO container_metrics (
        id, 
        name, 
        cpu_usage, 
        memory_usage, 
        memory_limit, 
        network_rx_speed,
        network_tx_speed,
        disk_read_speed,
        disk_write_speed,
        notifications_sent,
        status, 
        last_update
      )
      VALUES (
        @id, 
        @name, 
        @cpu_usage, 
        @memory_usage, 
        @memory_limit,
        @network_rx_speed,
        @network_tx_speed,
        @disk_read_speed,
        @disk_write_speed,
        @notifications_sent,
        @status, 
        CURRENT_TIMESTAMP
      )
      ON CONFLICT(id) DO UPDATE SET
        cpu_usage = excluded.cpu_usage,
        memory_usage = excluded.memory_usage,
        memory_limit = excluded.memory_limit,
        network_rx_speed = excluded.network_rx_speed,
        network_tx_speed = excluded.network_tx_speed,
        disk_read_speed = excluded.disk_read_speed,
        disk_write_speed = excluded.disk_write_speed,
        notifications_sent = excluded.notifications_sent,
        status = excluded.status,
        last_update = CURRENT_TIMESTAMP
    `);

    stmt.run(metrics);
  }

  public getMetrics(): ContainerMetrics[] {
    const allMetrics = this.db
      .prepare("SELECT * FROM container_metrics")
      .all() as ContainerMetrics[];

    const groupedMetrics = new Map<string, ContainerMetrics[]>();

    allMetrics.forEach((metric) => {
      const baseName = metric.name.split(".")[0];
      if (!groupedMetrics.has(baseName)) {
        groupedMetrics.set(baseName, []);
      }
      groupedMetrics.get(baseName)!.push(metric);
    });

    return Array.from(groupedMetrics.values()).map((metrics) => {
      const runningMetrics = metrics.filter((m) => m.status === "running");

      if (runningMetrics.length > 0) {
        runningMetrics.sort(
          (a, b) =>
            new Date(b.last_update).getTime() -
            new Date(a.last_update).getTime()
        );
        const latest = runningMetrics[0];

        const totalNotifications = metrics.reduce(
          (sum, m) => sum + (m.notifications_sent || 0),
          0
        );

        return {
          ...latest,
          name: latest.name.split(".")[0],
          notifications_sent: totalNotifications,
        };
      }

      metrics.sort(
        (a, b) =>
          new Date(b.last_update).getTime() - new Date(a.last_update).getTime()
      );

      const latest = metrics[0];
      const totalNotifications = metrics.reduce(
        (sum, m) => sum + (m.notifications_sent || 0),
        0
      );

      return {
        ...latest,
        name: latest.name.split(".")[0],
        notifications_sent: totalNotifications,
      };
    });
  }

  public async getNotificationCount(containerId: string): Promise<number> {
    const result = this.db
      .prepare("SELECT notifications_sent FROM container_metrics WHERE id = ?")
      .get(containerId) as { notifications_sent: number } | undefined;
    return result?.notifications_sent || 0;
  }

  public incrementNotifications(containerId: string): void {
    const container = this.db
      .prepare("SELECT name FROM container_metrics WHERE id = ?")
      .get(containerId) as { name: string } | undefined;

    if (container) {
      const baseName = container.name.split(".")[0];

      this.db
        .prepare(
          `
        UPDATE container_metrics 
        SET notifications_sent = notifications_sent + 1 
        WHERE id = (
          SELECT id 
          FROM container_metrics 
          WHERE name LIKE '${baseName}%' 
          ORDER BY last_update DESC 
          LIMIT 1
        )
      `
        )
        .run();
    }
  }

  public cleanUnmonitoredMetrics(monitoredNames: string[]): void {
    const placeholders = monitoredNames.map(() => "?").join(",");
    const sql = `DELETE FROM container_metrics WHERE name NOT IN (${placeholders})`;

    try {
      this.db.prepare(sql).run(monitoredNames);
    } catch (error) {
      console.error("Error cleaning unmonitored metrics:", error);
    }
  }
}

export const databaseService = new DatabaseService();
