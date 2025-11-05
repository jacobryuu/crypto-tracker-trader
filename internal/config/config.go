package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	MasterDatabaseURL string
	SlaveDatabaseURL  string
	EthereumNodeURL   string
}

func Load() (*Config, error) {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Error loading .env file, using system environment variables: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	masterDBURL := os.Getenv("MASTER_DATABASE_URL")
	slaveDBURL := os.Getenv("SLAVE_DATABASE_URL")
	ethereumNodeURL := os.Getenv("ETHEREUM_NODE_URL")

	return &Config{
		Port:              port,
		MasterDatabaseURL: masterDBURL,
		SlaveDatabaseURL:  slaveDBURL,
		EthereumNodeURL:   ethereumNodeURL,
	}, nil
}
