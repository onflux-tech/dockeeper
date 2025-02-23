import { NotificationService } from "../../config/environment";
import { INotificationService } from "./notification.interface";
import { EvolutionNotificationService } from "./evolution-notification.service";
import { WuzapiNotificationService } from "./wuzapi-notification.service";

export class NotificationFactory {
  static createNotificationService(
    service: NotificationService
  ): INotificationService {
    switch (service) {
      case "evolution":
        return new EvolutionNotificationService();
      case "wuzapi":
        return new WuzapiNotificationService();
      default:
        throw new Error(`Notification service ${service} not implemented`);
    }
  }
}
