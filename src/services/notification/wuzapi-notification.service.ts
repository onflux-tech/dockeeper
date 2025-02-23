import { config } from "../../config/environment";
import { INotificationService } from "./notification.interface";

interface WuzapiPayload {
  Phone: string;
  Body: string;
}

export class WuzapiNotificationService implements INotificationService {
  async sendNotification(message: string): Promise<void> {
    const { api, key, number } = config.notification;

    if (!api || !key || !number) {
      console.log("Wuzapi notification not configured");
      return;
    }

    const payload: WuzapiPayload = {
      Phone: number,
      Body: message,
    };

    try {
      const response = await fetch(api, {
        method: "POST",
        headers: {
          token: key,
          Accept: "application/json",
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const body = await response.text();
        throw new Error(`Notification error response: ${body}`);
      }

      console.log(`Notification sent via Wuzapi, status: ${response.status}`);
    } catch (error) {
      console.error("Error sending Wuzapi notification:", error);
    }
  }
}
