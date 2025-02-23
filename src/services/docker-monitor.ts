import Docker from "dockerode";
import { config } from "../config/environment";
import { notificationServiceEvolution } from "./notification-service-evolution";
import { notificationServiceMeow } from "./notification-service-meow";

interface DockerService {
  ID: string;
  Spec: {
    Name: string;
  };
}

interface DockerTask {
  Status: {
    State: string;
    Timestamp: string;
    Err?: string;
  };
}

export class DockerMonitor {
  private docker: Docker;
  private containerStates: Map<string, string> = new Map();
  private serviceStates: Map<string, string> = new Map();
  private notificationDebounce: Map<string, number> = new Map();

  constructor(docker: Docker) {
    this.docker = docker;
  }

  async startMonitoring(): Promise<void> {
    if (config.monitoring.healthChecks.length === 0) {
      console.log("No monitoring configured, skipping check");
      return;
    }

    console.log("Starting Docker events monitoring in real time.");
    console.log("Monitored containers:", config.monitoring.healthChecks);

    await this.initializeStates();

    const eventStream = await this.docker.getEvents({
      filters: {
        type: ["container", "service"],
        event: [
          "die",
          "stop",
          "kill",
          "start",
          "update",
          "remove",
          "create",
          "health_status",
          "shutdown",
          "destroy",
          "pause",
          "unpause",
        ],
      },
    });

    eventStream.on("data", async (buffer) => {
      const event = JSON.parse(buffer.toString());
      await this.handleEvent(event);
    });

    eventStream.on("error", (err) => {
      console.error("Error in Docker event stream:", err);
    });

    if (config.docker.isSwarm) {
      setInterval(() => this.checkServicesHealth(), 5000);
    }
  }

  private async initializeStates(): Promise<void> {
    try {
      const containers = await this.docker.listContainers({ all: true });
      for (const container of containers) {
        const name = container.Names[0]?.slice(1);
        if (name && config.monitoring.healthChecks.includes(name)) {
          this.containerStates.set(container.Id, container.State);
        }
      }

      if (config.docker.isSwarm) {
        const services = await this.docker.listServices();
        for (const service of services) {
          const typedService = service as DockerService;
          const serviceName = typedService.Spec?.Name;
          if (
            serviceName &&
            config.monitoring.healthChecks.includes(serviceName)
          ) {
            this.serviceStates.set(serviceName, "running");
          }
        }
      }
    } catch (error) {
      console.error("Error initializing states:", error);
    }
  }

  private async checkServicesHealth(): Promise<void> {
    try {
      const services = await this.docker.listServices();

      for (const service of services) {
        const typedService = service as DockerService;
        const serviceName = typedService.Spec?.Name;

        if (
          !serviceName ||
          !config.monitoring.healthChecks.includes(serviceName)
        ) {
          continue;
        }

        const tasks = await this.docker.listTasks({
          filters: { service: [serviceName] },
        });

        const failedTasks = tasks.filter((task: DockerTask) => {
          const isFailedState = [
            "failed",
            "rejected",
            "shutdown",
            "orphaned",
            "remove",
          ].includes(task.Status.State);
          if (!isFailedState) return false;

          const timestamp = new Date(task.Status.Timestamp).getTime();
          return Date.now() - timestamp < 10000;
        });

        const latestFailedTask = failedTasks.reduce((latest, current) => {
          if (!latest || !latest.Status.Timestamp) return current;
          if (!current.Status.Timestamp) return latest;
          return new Date(current.Status.Timestamp) >
            new Date(latest.Status.Timestamp)
            ? current
            : latest;
        }, failedTasks[0]);

        if (latestFailedTask) {
          const previousState = this.serviceStates.get(serviceName);
          const lastNotification =
            this.notificationDebounce.get(serviceName) || 0;
          const now = Date.now();

          if (previousState === "running" && now - lastNotification > 10000) {
            this.notificationDebounce.set(serviceName, now);
            this.serviceStates.set(serviceName, latestFailedTask.Status.State);

            await this.sendServiceAlert(
              serviceName,
              latestFailedTask.Status.State,
              latestFailedTask.Status.Err
            );
          }
        } else {
          this.serviceStates.set(serviceName, "running");
        }
      }
    } catch (error) {
      console.error("Error checking service health:", error);
    }
  }

  private async sendServiceAlert(
    serviceName: string,
    state: string,
    error?: string
  ): Promise<void> {
    let message = `🚨 *Service Alert*\n\n`;
    message += `📦 *Service:* ${serviceName}\n`;
    message += `📊 *Status:* ${state}\n`;
    message += `⏰ *Date/Time:* ${this.formatDateTime(new Date())}`;

    if (error) {
      message += `\n❌ *Error:* ${error}`;
    }

    if (config.notificationService === "evolution") {
      await notificationServiceEvolution.sendNotification(message);
    } else if (config.notificationService === "meow") {
      await notificationServiceMeow.sendNotification(message);
    }
  }

  private async handleEvent(event: any): Promise<void> {
    if (event.Type === "service") {
      await this.handleServiceEvent(event);
    } else if (event.Type === "container") {
      await this.handleContainerEvent(event);
    }
  }

  private async handleServiceEvent(event: any): Promise<void> {
    const serviceName = event.Actor?.Attributes?.name;
    if (!serviceName || !config.monitoring.healthChecks.includes(serviceName)) {
      return;
    }

    const failureEvents = ["remove", "die", "kill", "shutdown", "failed"];
    if (failureEvents.includes(event.Action)) {
      const lastNotification = this.notificationDebounce.get(serviceName) || 0;
      const now = Date.now();

      if (now - lastNotification > 10000) {
        this.notificationDebounce.set(serviceName, now);
        await this.sendServiceAlert(serviceName, event.Action);
      }
    }
  }

  private async handleContainerEvent(event: any): Promise<void> {
    const name = event.Actor?.Attributes?.name;
    const containerId = event.Actor?.ID;

    if (
      !name ||
      !containerId ||
      !config.monitoring.healthChecks.includes(name)
    ) {
      return;
    }

    try {
      const container = await this.docker.getContainer(containerId).inspect();
      const currentState = container.State.Status;
      const previousState = this.containerStates.get(containerId);

      this.containerStates.set(containerId, currentState);

      if (currentState !== "running" && previousState === "running") {
        let message = `🚨 *Container Alert*\n\n`;
        message += `📦 *Container:* ${name}\n`;
        message += `📊 *Status:* ${currentState}\n`;
        message += `⏰ *Date/Time:* ${this.formatDateTime(new Date())}`;

        if (container.State.Error) {
          message += `\n❌ *Error:* ${container.State.Error}`;
        }

        if (config.notificationService === "evolution") {
          await notificationServiceEvolution.sendNotification(message);
        } else if (config.notificationService === "meow") {
          await notificationServiceMeow.sendNotification(message);
        }
      }
    } catch (err) {
      console.error(`Error processing container event ${containerId}:`, err);
    }
  }

  private formatDateTime(date: Date): string {
    return date.toLocaleString("pt-BR", {
      timeZone: "America/Sao_Paulo",
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
    });
  }
}
