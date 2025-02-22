# DocKeeper

[![License](https://img.shields.io/github/license/onflux-tech/dockeeper)](LICENSE)
[![Release](https://img.shields.io/github/v/release/onflux-tech/dockeeper)](https://github.com/onflux-tech/dockeeper/releases)
[![Deploy Status](https://github.com/onflux-tech/dockeeper/actions/workflows/deploy.yml/badge.svg)](https://github.com/onflux-tech/dockeeper/actions/workflows/deploy.yml)

DocKeeper is a robust Docker maintenance and monitoring tool that helps keep your Docker environment clean and healthy while providing real-time notifications about container and service status changes.

## Features

- 🧹 **Automatic Cleanup**
  - Removes stopped containers
  - Cleans unused images
  - Purges build cache
- 📊 **Real-time Monitoring**
  - Monitors container state changes
  - Tracks service health in Swarm mode or Standalone
  - Sends instant notifications via WhatsApp (Evolutio API)
- 🔄 **Periodic Maintenance**
  - Configurable cleanup intervals
  - Automated maintenance cycles
- 🌐 **HTTP API**
  - Health check endpoint
  - Manual maintenance trigger
  - Status monitoring

## Technologies

- **[Node.js](https://nodejs.org/)** (v20+) - JavaScript runtime
- **[TypeScript](https://www.typescriptlang.org/)** - Type-safe JavaScript
- **[Dockerode](https://github.com/apocas/dockerode)** - Docker API integration
- **[Express](https://expressjs.com/)** - HTTP server framework
- **[Evolution API](https://github.com/evolution-api/evolution-api)** - WhatsApp notification system
- **[Docker](https://www.docker.com/)** - Containerization platform

## Environment Variables

| Variable                 | Description                                          | Default      |
| ------------------------ | ---------------------------------------------------- | ------------ |
| `CLEANUP_INTERVAL_HOURS` | Hours between cleanup cycles                         | 24           |
| `HEALTH_CHECKS`          | Containers/services to monitor (semicolon-separated) | ""           |
| `EVO_INSTANCE`           | Evolution API instance URL                           | ""           |
| `EVO_APIKEY`             | Evolution API key                                    | ""           |
| `EVO_NUMBER`             | WhatsApp number for notifications                    | ""           |
| `DOCKER_MODE`            | Deployment mode (standalone/swarm)                   | "standalone" |

## Installation

### Docker Standalone

```bash
# Pull the image
docker pull ghcr.io/onflux-tech/dockeeper:latest

# Run the container
docker run -d \
  --name dockeeper \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e CLEANUP_INTERVAL_HOURS=24 \
  -e HEALTH_CHECKS=container1;container2 \
  -e EVO_INSTANCE=https://your-evo-api.com \
  -e EVO_APIKEY=your-api-key \
  -e EVO_NUMBER=your-number \
  ghcr.io/onflux-tech/dockeeper:latest
```

### Docker Swarm

```yaml
version: "3.8"
services:
  dockeeper:
    image: ghcr.io/onflux-tech/dockeeper:latest
    environment:
      - CLEANUP_INTERVAL_HOURS=24
      - HEALTH_CHECKS=service_service1;service_service2
      - EVO_INSTANCE=https://your-evo-api.com
      - EVO_APIKEY=your-api-key
      - EVO_NUMBER=your-number
      - DOCKER_MODE=swarm
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    deploy:
      mode: global
```

Deploy with:

```bash
docker stack deploy --detach=true --prune --resolve-image always -c docker-compose.yml dockeeper
```

> Docker Compose examples with Traefik in repository root

## API Endpoints

### Health Check

```bash
GET /health
```

Returns system status information.

### Manual Cleanup

```bash
POST /run
```

Triggers immediate maintenance cycle.

## Building from Source

```bash
# Clone the repository
git clone https://github.com/onflux-tech/dockeeper.git

# Install dependencies
npm install

# Build the project
npm run build

# Start in development mode
npm run dev

# Start in production mode
npm start
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow TypeScript best practices
- Write clean, documented code
- Add tests for new features
- Update documentation as needed
- Follow existing code style

## License

This project is licensed under the GNU Affero General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Author

DocKeeper is maintained by [OnFlux Technologies](https://github.com/onflux-tech).

## Acknowledgments

- Docker and the Docker community
- Evolution API developers
- All contributors to this project
