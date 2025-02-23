import { config } from "../config/environment";

interface NotificationPayload {
  Phone: string;
  Body: string;
}

class NotificationServiceMeow {
  async sendNotification(message: string): Promise<void> {
    const { instance, apiKey, number } = config.meow;

    if (!instance || !apiKey || !number) {
      console.log("Meow notification not configured");
      return;
    }

    const payload: NotificationPayload = {
      Phone: number,
      Body: message,
    };

    try {
      const response = await fetch(instance, {
        method: "POST",
        headers: {
          token: apiKey,
          "Accept": "application/json",
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

export const notificationServiceMeow = new NotificationServiceMeow();
