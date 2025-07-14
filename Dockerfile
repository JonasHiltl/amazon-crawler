FROM golang:1.24.2 AS build

RUN mkdir /app

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /app/amzn-crawler main.go

# Python is required to install/start camoufox
FROM python:3.12-slim

RUN apt-get update; apt-get install -y --no-install-recommends fonts-liberation xauth xvfb libgtk-3-0 libx11-xcb1 libasound2; rm -rf /var/lib/apt/lists/*

# Install Camoufox
RUN pip3 install --no-cache-dir camoufox[geoip] --break-system-packages && \
    playwright install-deps && \
    python3 -m camoufox fetch && \
    rm -rf ./root/.cache/camoufox/fonts/macos ./root/.cache/camoufox/fonts/linux
# remove macos and linux fonts since we only use windows User-Agents in the crawler

WORKDIR /app

COPY --from=build /app/amzn-crawler .

RUN echo "PLAYWRIGHT_DRIVER_DIR=$(python3 -c 'import site; print(site.getsitepackages()[0] + "/playwright/driver")')" >> .env

CMD ["/app/amzn-crawler"]