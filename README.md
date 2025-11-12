# Home Server Status Reporter

[![CI](https://github.com/ivolong/home-server-status-reporter/actions/workflows/build.yml/badge.svg)](https://github.com/ivolong/home-server-status-reporter/actions/workflows/build.yml)

[![Go](https://img.shields.io/badge/Go-gray?logo=go&logoColor=white&style=flat)](https://go.dev/)

Simple, lightweight web application for monitoring system and service health on an Ubuntu server.

## Usage

1. Clone this repository.
1. Edit `config.json` with your server and service information.
   - You can configure simple healthchecks for web-based applications.
1. Run `docker compose up`, or run directly with Go.
1. `curl localhost:3000` to validate it is running. 
