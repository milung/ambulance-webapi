package main

import (
	"context"
	_ "embed"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/milung/ambulance-webapi/api"
	"github.com/milung/ambulance-webapi/internal/ambulance_wl"
	"github.com/milung/ambulance-webapi/internal/db_service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/technologize/otel-go-contrib/otelginmetrics"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// initialize OpenTelemetry instrumentations
func initTelemetry() (func(context.Context) error, error) {
	ctx := context.Background()
	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceNameKey.String("Ambulance WebAPI Service")),
		resource.WithAttributes(semconv.ServiceNamespaceKey.String("WAC Hospital")),
		resource.WithSchemaURL(semconv.SchemaURL),
		resource.WithContainer(),
	)

	if err != nil {
		return nil, err
	}

	metricExporter, err := prometheus.New()
	if err != nil {
		return nil, err
	}
	metricProvider := metric.NewMeterProvider(metric.WithReader(metricExporter), metric.WithResource(res))
	otel.SetMeterProvider(metricProvider)

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	// setup trace exporter, only otlp supported
	// see also https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/exporters/autoexport
	traceExportType := os.Getenv("OTEL_TRACES_EXPORTER")
	if traceExportType == "otlp" {
		log.Printf("OTLP trace exporter is configured")
		// we will configure exporter by using env variables defined
		// at https://opentelemetry.io/docs/concepts/sdk-configuration/otlp-exporter-configuration/
		traceExporter, err := otlptracegrpc.New(ctx)
		if err != nil {
			return nil, err
		}

		traceProvider := trace.NewTracerProvider(
			trace.WithResource(res),
			trace.WithSyncer(traceExporter))

		otel.SetTracerProvider(traceProvider)
		otel.SetTextMapPropagator(propagation.TraceContext{})
		// Shutdown function will flush any remaining spans
		return traceProvider.Shutdown, nil
	} else {
		// no otlp trace exporter configured
		noopShutdown := func(context.Context) error { return nil }
		return noopShutdown, nil
	}

}

func main() {
	log.Printf("Server started")

	port := os.Getenv("AMBULANCE_API_PORT")
	if port == "" {
		port = "8080"
	}

	environment := os.Getenv("AMBULANCE_API_ENVIRONMENT")
	if !strings.EqualFold(environment, "production") { // case insensitive comparison
		gin.SetMode(gin.DebugMode)
	}
	engine := gin.New()
	engine.Use(gin.Recovery())

	// setup telemetry
	shutdown, err := initTelemetry()
	if err != nil {
		log.Fatalf("Failed to initialize telemetry: %v", err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	// instrument gin engine
	engine.Use(
		otelginmetrics.Middleware(
			"Ambulance WebAPI Service",
			// Custom attributes
			otelginmetrics.WithAttributes(func(serverName, route string, request *http.Request) []attribute.KeyValue {
				return append(otelginmetrics.DefaultAttributes(serverName, route, request))
			}),
		),
		otelgin.Middleware("wl-webapi-server"),
	)

	// setup context update  middleware
	dbService := db_service.NewMongoService[ambulance_wl.Ambulance](db_service.MongoServiceConfig{})
	defer dbService.Disconnect(context.Background())
	engine.Use(func(ctx *gin.Context) {
		ctx.Set("db_service", dbService)
		ctx.Next()
	})

	// request routings
	ambulance_wl.AddRoutes(engine)

	// openapi spec endpoint
	engine.GET("/openapi", api.HandleOpenApi)

	// metrics endpoint
	promhandler := promhttp.Handler()
	engine.Any("/metrics", func(ctx *gin.Context) {
		promhandler.ServeHTTP(ctx.Writer, ctx.Request)
	})

	engine.Run(":" + port)
}
