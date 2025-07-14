# Amazon Crawler

This project is part of my Bachelor Thesis and includes building a distributed web crawler for extracting structured product data from [amazon.com](https://amazon.com). It depends on [Playwright Go](https://github.com/playwright-community/playwright-go) in combination with [Camoufox](https://camoufox.com) as the headless browser. 
The crawler begins with a predefined set of `SEED_URLS` and continuously discovers new product pages through search and category pages. URLs are queued in a PostgreSQL-backed job system to support scalable, distributed processing. Parsed product data is passed to a loosely coupled consumer module, with support for `stdout` and `opensearch` currently implemented.

## Getting Started
The crawler depends on `Camoufox`, `xvfb` and other system libraries to be installed on the system. The easiest way to get started is to use the included Docker setup.

### With Docker Compose
```bash
docker compose up -d
```
Afterwards visit the OpenSearch Dashboard at [http://localhost:5601](http://localhost:5601).

## License
This project is part of academic research and may not be used for commercial purposes without permission.
Please note: This crawler targets Amazon, which prohibits automated access in its Terms of Service. Use responsibly and at your own risk.