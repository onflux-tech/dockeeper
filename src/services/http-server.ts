import express from "express";

declare module "express-session" {
  interface SessionData {
    authenticated?: boolean;
  }
}
import Docker from "dockerode";
import path from "path";
import session from "express-session";
import SQLiteStoreFactory from "connect-sqlite3";
const SQLiteStore = SQLiteStoreFactory(session);
import { cleanExitedContainers } from "./container-cleaner";
import { cleanUnusedImages } from "./image-cleaner";
import { cleanBuildCache } from "./cache-cleaner";

interface StatusResponse {
  status: "ok" | "error";
  lastRun: Date | null;
  mode: string;
  running: boolean;
}

export class HttpServer {
  private app = express();
  private server: any;
  private lastRun: Date | null = null;
  private isRunning = false;

  constructor(private docker: Docker) {
    this.setupRoutes();
  }

  private validateToken(
    req: express.Request,
    res: express.Response,
    next: express.NextFunction
  ): void {
    const token = req.query.token as string;
    const apiToken = process.env.API_TOKEN;

    if (!apiToken) {
      res.status(500).json({ message: "API token not configured" });
      return;
    }

    if (!token || token !== apiToken) {
      res.status(401).json({ message: "Unauthorized - Invalid token" });
      return;
    }

    next();
  }

  private setupRoutes(): void {
    this.app.set("view engine", "ejs");

    this.app.use(
      session({
        store: new SQLiteStore({
          db: "sessions.sqlite",
          dir: path.join(process.cwd(), "data"),
        }) as unknown as session.Store,
        secret: process.env.API_TOKEN || "your-secure-token-here",
        resave: false,
        saveUninitialized: false,
        cookie: {
          secure: false,
          httpOnly: true,
          maxAge: 1000 * 60 * 60 * 24,
        },
        name: "dockeeper.sid",
      })
    );

    this.app.use(express.urlencoded({ extended: true }));

    const authMiddleware = (
      req: express.Request,
      res: express.Response,
      next: express.NextFunction
    ) => {
      const publicPaths = ["/login", "/health", "/run"];

      if (req.path === "/health" || req.path === "/run") {
        return this.validateToken(req, res, next);
      }

      if (req.path === "/" || req.path === "/login") {
        if (req.session.authenticated) {
          return res.redirect("/manager");
        }
        if (req.path === "/") {
          return res.redirect("/login");
        }
      }

      if (!publicPaths.includes(req.path) && !req.session.authenticated) {
        return res.redirect("/login");
      }

      next();
    };

    this.app.use(authMiddleware);

    this.app.get("/", (req, res) => {
      if (req.session.authenticated) {
        res.redirect("/manager");
      } else {
        res.redirect("/login");
      }
    });

    this.app.get("/health", (_, res) => {
      const status: StatusResponse = {
        status: "ok",
        lastRun: this.lastRun,
        mode: process.env.DOCKER_MODE || "standalone",
        running: this.isRunning,
      };
      res.json(status);
    });

    this.app.post("/run", async (_, res) => {
      if (this.isRunning) {
        return res.status(409).json({
          message: "Maintenance is already running",
        });
      }

      this.runMaintenance();
      res.json({
        message: "Maintenance started in background",
      });
    });

    this.app.get("/login", (req, res) => {
      if (req.session.authenticated) {
        return res.redirect("/manager");
      }
      res.render("login", { error: null });
    });

    this.app.post("/login", (req, res) => {
      const { username, password } = req.body;

      if (
        username === process.env.ADMIN_USER &&
        password === process.env.ADMIN_PASSWORD
      ) {
        req.session.authenticated = true;
        req.session.save((err) => {
          if (err) {
            console.error("Session save error:", err);
            return res.status(500).render("login", {
              error: "Error saving session",
            });
          }
          res.redirect("/manager");
        });
      } else {
        res.status(401).render("login", {
          error: "Invalid username or password",
        });
      }
    });

    this.app.get("/logout", (req, res) => {
      req.session.destroy(() => {
        res.redirect("/login");
      });
    });

    const viewsPath =
      process.env.NODE_ENV === "production"
        ? path.join(__dirname, "..", "views")
        : path.join(__dirname, "..", "..", "src", "views");

    this.app.set("views", viewsPath);

    this.app.get("/manager", (req, res) => {
      if (!req.session.authenticated) {
        return res.redirect("/login");
      }
      res.render("manager");
    });
  }

  private async runMaintenance(): Promise<void> {
    if (this.isRunning) return;

    this.isRunning = true;
    try {
      console.log("Starting Docker cleanup via API...");
      await cleanExitedContainers(this.docker);
      await cleanUnusedImages(this.docker);
      await cleanBuildCache(this.docker);
      this.lastRun = new Date();
      console.log("Docker cleanup via API completed.");
    } catch (error) {
      console.error("Error during maintenance:", error);
    } finally {
      this.isRunning = false;
    }
  }

  public getServer() {
    return this.server;
  }

  public start(): void {
    const port = process.env.PORT || 5000;
    this.server = this.app.listen(port, () => {
      console.log(`HTTP server started on port ${port}`);
    });
  }

  public stop(): void {
    if (this.server) {
      this.server.close();
    }
  }
}
