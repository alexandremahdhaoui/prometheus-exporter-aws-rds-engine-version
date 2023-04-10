# prometheus-exporter-aws-rds-engine-version

This tool exports metrics related to RDS clusters and their instances for Prometheus. It verifies the engine version of
each cluster and instance and creates a metric indicating if the current version is deprecated or available. The metric
is updated at a configurable interval.

## Prerequisites

### IAM Roles & Policies

This exporter will use IRSA to assume a role on which we'll attach the following policy.
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "",
            "Effect": "Allow",
            "Action": [
                "rds:DescribeDBInstances",
                "rds:DescribeDBClusters"
            ],
            "Resource": "*"
        }
    ]
}
```

## Installation

```bash
git clone https://github.com/alexandremahdhaoui/prometheus-exporter-aws-rds-engine-version
cd prometheus-exporter-aws-rds-engine-version
```

## Build the application.

```bash
go build
```

## Configuration
The exporter requires the following environment variables:

| Name                                | Description                                                       | 
|-------------------------------------|-------------------------------------------------------------------|
| `EXPORTER_AWS_API_INTERVAL_SECONDS` | the interval in seconds to update the metrics (recommended: 300). |
| `EXPORTER_SERVER_PORT`              | the port number that the server listens on (recommended: 2112).   |

## Usage

Start the exporter by running the following command:
```bash
./prometheus-exporter-aws-rds-engine-version
```

Access the metrics on the server's endpoint:
```bash
curl http://localhost:2112/metrics
```

### Metrics


| Name                              | Description                                          | Tags                                             | 
|-----------------------------------|------------------------------------------------------|--------------------------------------------------|
| aws_custom_rds_version_available  | Number of instances running an available rds version | "cluster_identifier", "engine", "engine_version" | 
| aws_custom_rds_version_deprecated | Number of instances running a deprecated rds version | "cluster_identifier", "engine", "engine_version" | 


## License
MIT License

## Contributing
Contributions are welcome. Feel free to open a pull request or an issue if you encounter any problems.

## Disclaimer

This project is a fork of [chapsy/aws-rds-engine-version-prometheus-exporter](https://github.com/chaspy/aws-rds-engine-version-prometheus-exporter).
