# crypto-tracker-trader

## Project Overview
This project is a crypto tracker and trader application that aims to provide integrated management and visualization of user assets from multiple cryptocurrency exchanges (Binance, Coinbase, OKX, Bybit, etc.) and DeFi protocols. It currently supports fetching and saving Ethereum (ETH) balance from the blockchain and provides a portfolio history.

## Features
- **Portfolio Management**: Track and visualize your cryptocurrency portfolio.
- **Ethereum Balance Fetcher**: Fetch and save ETH balance for a given Ethereum address.
- **API Endpoints**:
    - `/health`: Check the health of the application.
    - `/api/v1/portfolio/history`: Get the historical portfolio snapshots.
    - `/api/v1/blockchain/fetch-eth-balance/:address`: Fetch and save ETH balance for a specific Ethereum address.
- **Database**: Uses PostgreSQL to store portfolio data.

## Technologies Used
- **Backend**: Go (Gin Web Framework)
- **Database**: PostgreSQL
- **Blockchain Interaction**: go-ethereum (for Ethereum node interaction)
- **Dependency Management**: Go Modules
- **Containerization**: Docker, Docker Compose

## Getting Started

### Prerequisites
- Go (version 1.25.2 or higher)
- Docker and Docker Compose
- PostgreSQL client (optional, for direct database interaction)
- An Ethereum node URL (e.g., Infura, Alchemy, or a local node)

### Setup

1.  **Clone the repository**:
    ```bash
    git clone https://github.com/your-username/crypto-tracker-trader.git
    cd crypto-tracker-trader
    ```

2.  **Environment Variables**:
    Create a `.env` file in the root directory with the following content:
    ```
    MASTER_DATABASE_URL="postgres://user:password@host:port/database?sslmode=disable"
    ETHEREUM_NODE_URL="YOUR_ETHEREUM_NODE_URL"
    PORT="8080"
    ```
    Replace `user`, `password`, `host`, `port`, `database` with your PostgreSQL credentials and `YOUR_ETHEREUM_NODE_URL` with your Ethereum node endpoint. If using `docker-compose`, the `MASTER_DATABASE_URL` will be set by `docker-compose.yml`.

3.  **Run with Docker Compose (Recommended for Development)**:
    This will set up the PostgreSQL database and run the application.
    ```bash
    docker-compose up --build
    ```
    The application will be accessible at `http://localhost:8080`.

4.  **Run Locally (without Docker Compose)**:

    a.  **Start PostgreSQL**:
        Ensure you have a PostgreSQL instance running and accessible via the `MASTER_DATABASE_URL` provided in your `.env` file.

    b.  **Install Go Dependencies**:
        ```bash
        go mod tidy
        ```

    c.  **Run Database Migrations**:
        (Assuming you have `migrate` CLI tool installed or similar)
        ```bash
        # Example: Using a migration tool
        # migrate -path deployments/migrations -database "YOUR_MASTER_DATABASE_URL" up
        ```
        *Note: The project currently includes a migration file `0001_create_portfolio_tables.up.sql`.*

    d.  **Run the application**:
        ```bash
        go run cmd/server/main.go
        ```
        The application will be accessible at `http://localhost:8080`.

### Local Development with Foundry

If you are developing smart contracts or need a local Ethereum environment for testing, you can use Foundry.

1.  **Install Foundry**:
    Follow the official Foundry installation guide: [https://book.getfoundry.sh/getting-started/installation](https://book.getfoundry.sh/getting-started/installation)

2.  **Start a Local Ethereum Node (Anvil)**:
    Open a new terminal and run Anvil:
    ```bash
    anvil
    ```
    This will start a local Ethereum node, typically on `http://127.0.0.1:8545`.

3.  **Configure `ETHEREUM_NODE_URL`**:
    Update your `.env` file to point to your local Anvil instance:
    ```
    ETHEREUM_NODE_URL="http://127.0.0.1:8545"
    ```
    Now, when you run the application, it will connect to your local Foundry Anvil node.

## API Endpoints

### Health Check
`GET /health`
Returns: `{"status": "ok"}`

### Get Portfolio History
`GET /api/v1/portfolio/history`
Returns a JSON array of portfolio snapshots.

### Fetch and Save ETH Balance
`POST /api/v1/blockchain/fetch-eth-balance/:address`
Replace `:address` with a valid Ethereum address.
Returns: `{"message": "ETH balance fetched and saved successfully"}` or an error.

## Project Structure
```
.
├── cmd/                # Main application entry points
│   └── server/         # Server application
│       └── main.go     # Main server file
├── configs/            # Configuration files
├── deployments/        # Deployment related files (Docker, Docker Compose, Migrations)
├── internal/           # Internal packages and business logic
│   ├── api/            # API handlers
│   ├── config/         # Application configuration loading
│   ├── model/          # Data models
│   ├── service/        # Business logic services
│   └── store/          # Database interaction (PostgreSQL)
├── go.mod              # Go module definition
├── go.sum              # Go module checksums
├── LICENSE             # Project license
├── Makefile            # Makefile for common tasks
├── README.md           # Project README
└── scripts/            # Utility scripts
```

## Contributing
(Add guidelines for contributing if applicable)

## License
This project is licensed under the [LICENSE_NAME] - see the [LICENSE](LICENSE) file for details.
