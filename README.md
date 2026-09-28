# gator

Gator is a command-line RSS feed aggregator written in Go and backed by PostgreSQL. It allows users to register, follow RSS feeds across the web, aggregate new posts on an automated background ticker, and view a personalized feed of articles.

## Prerequisites

Ensure the following tools are installed:
* Go (1.22+)
* PostgreSQL running locally or accessible via network

## Installation

Install the CLI binary into your Go bin directory:

go install github.com/Phen0mPhil/blog-aggregator@latest

Make sure your bin directory is in your system PATH:

export PATH=$PATH:$(go env GOPATH)/bin

## Configuration

Gator reads configuration from ~/.gatorconfig.json in your home directory.

Create the file with your local database URL:

{
  "db_url": "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable",
  "current_user_name": ""
}

Run your Goose schema migrations against your database before using the app.

## CLI Commands

* gator register <name> - Register and log in as a new user
* gator login <name> - Switch to an existing user
* gator users - List all registered users
* gator addfeed <name> <url> - Add and auto-follow a new RSS feed
* gator feeds - Print all tracked feeds and creators
* gator follow <url> - Follow a feed for the current user
* gator following - List feeds the current user follows
* gator unfollow <url> - Unfollow a feed for the current user
* gator agg <duration> - Run continuous background aggregation (e.g. 1m)
* gator browse [limit] - Show posts for followed feeds (default limit 2)
* gator reset - Clear all database tables
