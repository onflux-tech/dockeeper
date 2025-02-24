# Stage 1: Build
FROM node:20-alpine AS builder

WORKDIR /dockeeper

COPY package*.json ./
RUN npm ci

COPY . .
RUN npm run build

# Stage 2: Produção
FROM node:20-alpine

ENV TZ=America/Sao_Paulo
ENV NODE_ENV=production

RUN apk add --no-cache tzdata && \
    cp /usr/share/zoneinfo/$TZ /etc/localtime && \
    echo $TZ > /etc/timezone

WORKDIR /dockeeper

COPY --from=builder /dockeeper/dist ./dist
COPY --from=builder /dockeeper/package*.json ./

RUN npm ci --only=production

RUN mkdir -p data

EXPOSE 5000

CMD ["npm", "start"]
