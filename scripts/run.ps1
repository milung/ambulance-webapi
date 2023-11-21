param (
    $command
)

if (-not $command)  {
    $command = "start"
}

$env:AMBULANCE_API_ENVIRONMENT="Development"
$env:AMBULANCE_API_PORT="8080"
$env:AMBULANCE_API_MONGODB_USERNAME="root"
$env:AMBULANCE_API_MONGODB_PASSWORD="neUhaDnes"

# get script file folder
$ProjectRoot = "${PSScriptRoot}/.."

function mongo {
    
    docker compose --file ${ProjectRoot}/deployments/docker-compose/compose.yaml $args
}

switch ($command) {
    "help" {
        echo "Usage: run.ps1 [command]"
        echo ""
        echo "Commands:"
        echo "  start: Starts the ambulance-api-service"
        echo "  build: Builds the ambulance-api-service"
        echo "  test:  Runs all unit tests"
        echo "  mongo: Starts a mongodb instance for development"
        echo "  docker: Builds a docker image"
        echo "  openapi: Generates the openapi client"
    }
    "openapi" {
        docker run --rm -ti  -v ${ProjectRoot}:/local openapitools/openapi-generator-cli generate -c /local/scripts/generator-cfg.yaml 
    }
    "start" {
        try {
            mongo up --detach
            go run ${ProjectRoot}/cmd/ambulance-api-service
        } finally {
            mongo down
        }
    }
    "build" {
        go build -o ${ProjectRoot}/bin/ambulance-api-service ${ProjectRoot}/cmd/ambulance-api-service
    }
    "test" {
        go test -v ./...
    }
    "mongo" {
       mongo up
    }
    "docker" {
        docker build -t pfx/ambulance-wl-webapi -f ${ProjectRoot}/build/docker/Dockerfile . 
    }
    default {
        throw "Unknown command:"  + $command
    }
}

