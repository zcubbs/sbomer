# SBOMer

SBOMer is a Go-based tool for generating Software Bill of Materials (SBOM) for GitLab projects (eventually others). It provides an automated way to fetch projects from GitLab groups and generate SBOMs using Syft.

## Features

- **Group-Based Project Fetching**: Recursively fetch projects from specified GitLab groups and their subgroups
- **Topic-Based Filtering**: Skip projects with specific topics using exclude_topics configuration
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
  schedule: "once"     # once or cron format "seconds minutes hours days months days_of_the_week"
  batch_size: 10
  cool_off_secs: 5
  group_ids:
    - "your-group-id"  # Optional: Specify GitLab group IDs to fetch from
  include_topics:      # Optional: Include only projects with these topics
    - "sbomer"
  exclude_topics:      # Optional: Skip projects with these topics
    - "skip-sbom"
    - "no-sbom"

syft:
  syft_bin_path: bin/syft.exe
```

### Topic-Based Filtering

You can exclude projects from SBOM generation by adding specific topics to them in GitLab and listing those topics in the `exclude_topics` configuration. This is useful for:
- Skipping projects that don't need SBOMs
- Excluding test or template repositories
- Managing large groups of repositories efficiently

For example, if you add the topic "skip-sbom" to a GitLab project and include it in the `exclude_topics` list, that project will be automatically skipped during fetching.

## Environment Variables

- `SBOMER_GITLAB_TOKEN`: GitLab API token
- `SBOMER_DB_URL`: Database connection string
- `SBOMER_GITLAB_HOST`: GitLab host (default: gitlab.com)
- `SBOMER_GITLAB_SCHEME`: GitLab scheme (default: https)
- `SBOMER_FETCHER_EXCLUDE_TOPICS`: Comma-separated list of topics to exclude

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
