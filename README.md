# reputer

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

> **This project is archived.** Active development has moved to **[DevTrace](https://devtrace.thingz.io/)** — a hosted service that builds on the ideas explored in `reputer` and goes much further.

## Use DevTrace instead

[**DevTrace**](https://devtrace.thingz.io/) is a contributor trust scoring platform for open source. It's the spiritual successor to `reputer` — same core question ("can I trust this contributor?"), production-grade answer.

**What you get:**

- **23 signals across 5 weighted categories** produce a transparent trust score and letter grade for any GitHub contributor
- **AI risk narratives** explain *why* a contributor was flagged — not just a raw number
- **Behavioral analysis** detects burst-vanish patterns, velocity anomalies, and synthetic profiles
- **Bot & AI-generated contribution detection** as distinct signals
- **License footprint** across a contributor's repos
- **GitHub Action** that gates PR merges on contributor trust
- **REST API** for embedding scores into internal systems
- **Compliance mapping** to 8 of 20 NIST SSDF practices

**It's free.** The Free plan covers contributor scoring, 30-day history, and 60 req/hour. During beta, the **Pro plan is also free** — AI risk summaries, 365-day history, 1000 req/hour, 10 API keys, and SSDF + EU CRA compliance reports.

Get started: **[devtrace.thingz.io](https://devtrace.thingz.io/)** — GitHub login + one-click app install.

## About this repository

`reputer` was a CLI tool that calculated contributor reputation scores using a v3 risk-weighted categorical model. It pioneered the scoring approach now used in DevTrace, but is no longer maintained. The code remains here for reference under the [Apache 2.0 License](LICENSE).

If you have an existing dependency on the CLI, releases remain available on the [releases](https://github.com/mchmarny/reputer/releases) page, but you should migrate to DevTrace for ongoing updates, additional signals, and a maintained GitHub Action.

## License

[Apache License 2.0](LICENSE)
