import { config } from "../../config/environment";
import { INotificationService } from "./notification.interface";

export class EvolutionNotificationService implements INotificationService {
  async sendNotification(message: string): Promise<void> {
    const { api, key, number } = config.notification;

    if (!api || !key || !number) {
      console.log("Evolution notification not configured");
      return;
    }

    try {
      const response = await fetch(api, {
        method: "POST",
        headers: {
          apikey: key,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          number,
          text: message,
        }),
      });

      if (!response.ok) {
        const body = await response.text();
        throw new Error(`Notification error response: ${body}`);
      }

      console.log(
        `Notification sent via Evolution API, status: ${response.status}`
      );
    } catch (error) {
      console.error("Error sending Evolution notification:", error);
    }
  }
}
