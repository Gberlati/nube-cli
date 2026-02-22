# ☁️ nube-cli - a CLI for managing Tienda Nube stores.

![GitHub Repo Banner](https://ghrb.waren.build/banner?header=%E2%98%81%EF%B8%8F+nube-cli&bg=0055D4-001554&color=FFFFFF&headerfont=Google+Sans&watermarkpos=bottom-right)
<!-- Created with GitHub Repo Banner by Waren Gonzaga: https://ghrb.waren.build -->

Fast, agentic and script-friendly CLI for managing Tienda Nube stores from the terminal. JSON-first output, with support for multiple stores.

## Features

- **Store** — get store info and general settings
- **Products** — list/search/get/create/update/delete products and variants; look up by SKU; bulk-update stock and price; manage product images
- **Categories** — list/get/create/update/delete categories; organize storefront navigation hierarchy
- **Customers** — list/search/get/create/update/delete customers; inspect contact info and purchase history
- **Orders** — list/search/get/create/update orders; open/close/cancel; view audit history; attach invoices; manage fulfillment orders and tracking events
- **Draft Orders** — create/confirm/delete draft orders from outside channels
- **Abandoned Checkouts** — list/get abandoned checkouts; apply coupons to recover carts
- **Coupons & Discounts** — list/get/create/update/delete coupons; define cart-level promotion and tier discount rules
- **Transactions** — list/get/create transactions per order; post events to drive payment state transitions (authorize, capture, refund, chargeback)
- **Shipping** — manage carriers and rate options; set up real-time shipping rates; manage fulfillment events per order
- **Locations** — list/get/create/update/delete store locations; set priorities and default; inspect inventory levels
- **Blog & Pages** — manage blog posts (create/publish/unpublish) and static store pages; upload images; manage SEO metadata
- **Metafields** — manage namespaced key-value metafields scoped to any resource (products, orders, customers, etc.)
- **Webhooks** — list/get/create/update/delete event subscriptions; handle GDPR mandatory webhooks
- **Billing** — manage app plans, subscriptions, and charges
- **FTP Support** - manage store themes by connecting via FTP. 
- **Multiple accounts** - manage multiple Tienda Nube stores simultaneously (with aliases)
- **Command allowlist** - restrict top-level commands for sandboxed/agent runs
- **Secure credential storage** using OS keyring or encrypted on-disk keyring (configurable)
- **Parseable output** - JSON mode for scripting and automation

## Installation

### Build from Source

```bash
git clone https://github.com/gberlati/nube-cli.git
cd nube-cli
make
```

Run:

```bash
./bin/nube-cli --help
```

## Quick Start

## Environment Variables

 - `NUBE_KEYRING_PASSWORD`: Password used as fallback to encrypt token if no OS Keyring available.  
 - `NUBE_JSON` - Default JSON output
 - `NUBE_PLAIN` - Default plain output
 - `NUBE_COLOR` - Color mode: `auto` (default), `always`, or `never`
 - `NUBE_ENABLE_COMMANDS` - Comma-separated allowlist of top-level commands (e.g., `orders, billing`)

## Security

## License

MIT

## Links

 - [Github Repository]()
 - [Tienda Nube Documentation](https://tiendanube.github.io/api-documentation)

## Credits

This project is inspired by Peter Steinberg's google CLI.
 - [gogcli](https://github.com/steipete/gogcli)
