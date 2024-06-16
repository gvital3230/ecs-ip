# Project ECS Services IPs

Service returns page with ECS clusters, services and IP addresses of services. 

For the most of cases default AWS credentials will be used, from the `~/.aws/credentials` file. 

Please refer this page [https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials](https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials) for AWS SDK for Go V2 if you need to set up credentials in a different way.


## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

Clone the repository

```bash
git clone https://github.com/gvital3230/ecs-ip.git
```

Prepare the environment variables, copy the `.env.example` file to `.env` and set the values that you need
```bash
cp .env.example .env
```

Run the application using make commands

```bash
make run
```

## Application parameters

These values can be set using environment variables or `.env` file.

| Parameter      | Default Value | Description                                         |
|----------------|---------------|-----------------------------------------------------|
| PORT           | 8080          | Port for the application to listen on               |
| ADMIN_PASSWORD |               | Password to page, user name is defaulted to `admin` |


## MakeFile

run all make commands with clean tests
```bash
make all build
```

build the application
```bash
make build
```

run the application
```bash
make run
```

live reload the application
```bash
make watch
```

clean up binary from the last build
```bash
make clean
```
