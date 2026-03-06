# Examples

## Health Endpoint for an HTTP Service

Combine the health aggregator with an HTTP handler to expose a `/health` endpoint:

```go
package main

import (
    "context"
    "encoding/json"
    "net/http"

    "digital.vasic.observability/pkg/health"
)

func main() {
    agg := health.NewAggregator()
    agg.Register("database", pingDatabase)
    agg.Register("redis", pingRedis)
    agg.RegisterOptional("search-index", pingElastic)

    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        report := agg.Check(r.Context())

        status := http.StatusOK
        if report.Status == "unhealthy" {
            status = http.StatusServiceUnavailable
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(status)
        json.NewEncoder(w).Encode(report)
    })

    http.ListenAndServe(":8080", nil)
}
```

## Correlation ID Propagation

Pass correlation IDs through context for request tracing across services:

```go
package main

import (
    "context"
    "net/http"

    "digital.vasic.observability/pkg/logging"
)

func main() {
    logger := logging.NewLogrusAdapter(nil)

    http.HandleFunc("/api/process", func(w http.ResponseWriter, r *http.Request) {
        // Extract or generate correlation ID
        corrID := r.Header.Get("X-Correlation-ID")
        if corrID == "" {
            corrID = "gen-" + r.RemoteAddr
        }

        // Store in context
        ctx := logging.ContextWithCorrelationID(r.Context(), corrID)

        // Create a logger enriched from context
        reqLogger := logger.WithCorrelationID(corrID)
        reqLogger.Info("Processing request")

        // Pass context to downstream functions
        processOrder(ctx, reqLogger)

        w.WriteHeader(http.StatusOK)
    })

    http.ListenAndServe(":8080", nil)
}

func processOrder(ctx context.Context, logger logging.Logger) {
    // Correlation ID is available throughout the call chain
    corrID := logging.CorrelationIDFromContext(ctx)
    logger.WithField("step", "validation").Info("Validating order")
}
```

## ClickHouse Analytics with Graceful Degradation

Track events to ClickHouse, falling back to no-op when unavailable:

```go
package main

import (
    "context"
    "fmt"

    "digital.vasic.observability/pkg/analytics"
    "github.com/sirupsen/logrus"
)

func main() {
    logger := logrus.New()

    // If ClickHouse is unavailable, returns a NoOpCollector automatically
    collector := analytics.NewCollector(&analytics.ClickHouseConfig{
        Address:  "localhost:9000",
        Database: "analytics",
        Table:    "events",
    }, logger)
    defer collector.Close()

    // Track a single event
    err := collector.Track(context.Background(), &analytics.Event{
        Name:       "page_view",
        Properties: map[string]interface{}{"page": "/home", "user_id": 123},
    })
    if err != nil {
        logger.WithError(err).Warn("Failed to track event")
    }

    // Batch tracking for efficiency
    events := []*analytics.Event{
        {Name: "click", Properties: map[string]interface{}{"button": "signup"}},
        {Name: "click", Properties: map[string]interface{}{"button": "login"}},
    }
    collector.TrackBatch(context.Background(), events)

    // Query analytics (read-only SQL only)
    results, err := collector.ExecuteReadQuery(context.Background(),
        "SELECT count() FROM events WHERE name = 'page_view'")
    if err == nil {
        fmt.Println("Page views:", results)
    }
}
```
