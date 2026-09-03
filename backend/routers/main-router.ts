import { DockgeServer } from "../dockge-server";
import { Router } from "../router";
import express, { Express, Router as ExpressRouter } from "express";
import { ApiV1Router } from "./api-v1-router";

export class MainRouter extends Router {
    create(app: Express, server: DockgeServer): ExpressRouter {
        const router = express.Router();

        // Automation API is mounted through the existing primary router so
        // the upstream server bootstrap remains untouched and compatible.
        router.use(new ApiV1Router().create(app, server));

        router.get("/", (req, res) => {
            res.send(server.indexHTML);
        });

        // Robots.txt
        router.get("/robots.txt", async (_request, response) => {
            let txt = "User-agent: *\nDisallow: /";
            response.setHeader("Content-Type", "text/plain");
            response.send(txt);
        });

        return router;
    }

}
