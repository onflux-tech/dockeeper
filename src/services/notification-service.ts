import { config } from "../config/environment";

interface NotificationPayload {
  number: string;
  text: string;
}

class NotificationService {
  async sendNotification(message: string): Promise<void> {
    const { instance, apiKey, number } = config.evolution;

    if (!instance || !apiKey || !number) {
      console.log("Evolution notification not configured");
      return;
    }

    const payload: NotificationPayload = {
      number,
      text: message,
    };

    try {
      const response = await fetch(instance, {
        method: "POST",
        headers: {
          apikey: apiKey,
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const body = await response.text();
        throw new Error(`Notification error response: ${body}`);
      }

      console.log(`Notification sent, status: ${response.status}`);
    } catch (error) {
      console.error("Error sending notification:", error);
    }
  }
}

export const notificationService = new NotificationService();
