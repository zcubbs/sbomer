# SBOMer

SBOMer is a Go-based tool for generating Software Bill of Materials (SBOM) for GitLab projects (eventually others). It provides an automated way to fetch projects from GitLab groups and generate SBOMs using Syft.

## Features

- **Group-Based Project Fetching**: Recursively fetch projects from specified GitLab groups and their subgroups
- **Efficient Processing**: Process projects in batches with configurable batch sizes and cool-off periods
- **Message Queue Integration**: Uses RabbitMQ for reliable project processing
- **Database Storage**: Stores fetch statistics and operation logs in PostgreSQL
- **Syft Integration**: Generates SBOMs using Syft in CycloneDX JSON format

## Components

- **Fetcher**: Retrieves projects from GitLab and publishes them to RabbitMQ
- **Processor**: Clones repositories and generates SBOMs using Syft
- **Database**: Stores operational data and statistics
- **GitLab Client**: Handles GitLab API interactions and repository cloning

## Configuration

The application is configured via environment variables or a `config.yaml` file:

```yaml
app:
  log_level: info

gitlab:
  host: gitlab.com
  scheme: https
  token: "" # Set via SBOMER_GITLAB_TOKEN
  temp_dir: tmp/sbomer

database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  dbname: sbomer
  sslmode: disable

fetcher:
  schedule: "once"
  group_ids:
    - "your-group-id"
```

## Environment Variables

- `SBOMER_GITLAB_TOKEN`: GitLab API token
- `SBOMER_DB_URL`: Database connection string
- `SBOMER_GITLAB_HOST`: GitLab host (default: gitlab.com)
- `SBOMER_GITLAB_SCHEME`: GitLab scheme (default: https)

## Getting Started

1. Set up PostgreSQL database
2. Configure RabbitMQ
3. Set environment variables
4. Run the fetcher service:
   ```bash
   go run cmd/fetcher/main.go
   ```
5. Run the processor service:
   ```bash
   go run cmd/processor/main.go
   ```

## License

This project is licensed under the MIT License.
