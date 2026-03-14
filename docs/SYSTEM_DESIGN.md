# Discordiance — System Design

> Immutable architectural blueprint. Every implementation decision traces back to this document.

---

## 1. Principles

1. **Proto is the only source of truth.** Every type that crosses the wire is defined in proto. No hand-written Go structs, no hand-written TypeScript interfaces for API types. DB models are the sole exception.
2. **Every field is strictly typed.** No `map[string]string`, no `JSONMap`, no generic schema descriptors. Typed fields everywhere — proto, Go, TypeScript, DB.
3. **Interface-driven extensibility.** Platforms, reporters, and agent clients are Go interfaces. New implementations plug in without modifying core code.
4. **Minimal surface area.** Every file, function, and field must justify its existence. No speculative abstractions, no dead code, no "just in case" utilities.
5. **Enterprise patterns, startup velocity.** Clean layering, clear ownership boundaries, but no ceremony for ceremony's sake.

---

## 2. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Angular Frontend                         │
│                  (connect-web + PrimeNG)                         │
└──────────────────────────┬──────────────────────────────────────┘
                           │ Connect-RPC (HTTP/2)
┌──────────────────────────▼──────────────────────────────────────┐
│                      RPC Service Layer                          │
│  ProductService · AgentService · PlatformService                │
│  PipelineService · InsightService · ReporterService             │
│  HealthService                                                  │
├─────────────────────────────────────────────────────────────────┤
│                       Engine Layer                               │
│  Pipeline Runner · Ingestion · Classification · Dispatch         │
├──────────────┬──────────────────┬───────────────────────────────┤
│   Platform   │   Agent Client   │        Reporter               │
│   Adapters   │ (OpenAI-compat)  │        Handlers               │
│  ┌─────────┐ │  ┌────────────┐  │  ┌──────────┐ ┌───────────┐  │
│  │ Discord │ │  │  HTTP/SSE  │  │  │ Internal │ │  Webhook  │  │
│  │ Reddit  │ │  │  Client    │  │  │ (soft)   │ │  Email    │  │
│  │ Twitter │ │  │            │  │  │          │ │  GitHub   │  │
│  │  ...    │ │  └────────────┘  │  │          │ │   ...     │  │
│  └─────────┘ │                  │  └──────────┘ └───────────┘  │
├──────────────┴──────────────────┴───────────────────────────────┤
│                        Data Layer                                │
│                    GORM + SQLite                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. Data Flow

```
Platform Sources (Discord, Reddit, Twitter, ...)
         │
         ▼
┌──────────────────┐
│ Platform Adapter  │  Polls, streams, or crawls source
│ (ingestion)       │  Produces raw content
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Insight (RAW)     │  Stored in DB, queued for classification
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Agent Client      │  Sends insight(s) to OpenAI-compatible LLM
│ (classification)  │  Receives state + reasoning
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Insight           │  State updated: COLD / HOT / BURN
│ (classified)      │  Classification reason + agent ID recorded
└────────┬─────────┘
         │
         ▼
┌──────────────────┐
│ Reporter Dispatch │  Filters by state, platform, custom rules
│                   │  Routes to matching reporters
└────────┬─────────┘
         │
         ├──► Internal soft reporter (always, all non-RAW)
         ├──► Webhook reporter
         ├──► Email reporter
         └──► ... (any configured reporters)
```

---

## 4. Proto Contracts

All protos live under `proto/discordiance/v1/`. Package: `discordiance.v1`.

### 4.1 `common.proto` — Shared types

```protobuf
// Pagination for list RPCs.
message PaginationRequest {
  int32 page_size = 1;
  string page_token = 2;
}

message PaginationResponse {
  string next_page_token = 1;
  int32 total_count = 2;
}
```

### 4.2 `product.proto` — Products and context

```protobuf
// A product is the subject of a pipeline.
// Anything with a community: SaaS app, video game, OSS project, recipe, etc.
message Product {
  string id = 1;
  string name = 2;
  string description = 3;
  repeated ProductContext contexts = 4;
  google.protobuf.Timestamp created_at = 5;
  google.protobuf.Timestamp updated_at = 6;
}

// Typed context attached to a product for agent consumption.
message ProductContext {
  string id = 1;
  ProductContextType type = 2;
  string value = 3;   // URL, text, path — interpretation depends on type
  string label = 4;   // human-readable label
}

enum ProductContextType {
  PRODUCT_CONTEXT_TYPE_UNSPECIFIED = 0;
  PRODUCT_CONTEXT_TYPE_DESCRIPTION = 1;
  PRODUCT_CONTEXT_TYPE_URL = 2;
  PRODUCT_CONTEXT_TYPE_REPOSITORY = 3;
  PRODUCT_CONTEXT_TYPE_FILE = 4;
  PRODUCT_CONTEXT_TYPE_DOCUMENTATION = 5;
}

service ProductService {
  rpc CreateProduct(CreateProductRequest) returns (CreateProductResponse) {}
  rpc GetProduct(GetProductRequest) returns (GetProductResponse) {}
  rpc ListProducts(ListProductsRequest) returns (ListProductsResponse) {}
  rpc UpdateProduct(UpdateProductRequest) returns (UpdateProductResponse) {}
  rpc DeleteProduct(DeleteProductRequest) returns (DeleteProductResponse) {}
  rpc AddProductContext(AddProductContextRequest) returns (AddProductContextResponse) {}
  rpc RemoveProductContext(RemoveProductContextRequest) returns (RemoveProductContextResponse) {}
}
```

### 4.3 `agent.proto` — LLM agent configuration

```protobuf
// An agent is an LLM endpoint using the OpenAI-compatible API.
message Agent {
  string id = 1;
  string name = 2;
  string base_url = 3;       // OpenAI-compatible API base URL
  string model = 4;           // model identifier (e.g. "gpt-4o", "claude-sonnet-4-20250514")
  string api_key = 5;         // redacted in read responses
  int32 max_tokens = 6;
  double temperature = 7;
  google.protobuf.Timestamp created_at = 8;
  google.protobuf.Timestamp updated_at = 9;
}

service AgentService {
  rpc CreateAgent(CreateAgentRequest) returns (CreateAgentResponse) {}
  rpc GetAgent(GetAgentRequest) returns (GetAgentResponse) {}
  rpc ListAgents(ListAgentsRequest) returns (ListAgentsResponse) {}
  rpc UpdateAgent(UpdateAgentRequest) returns (UpdateAgentResponse) {}
  rpc DeleteAgent(DeleteAgentRequest) returns (DeleteAgentResponse) {}
  // Validates the endpoint is reachable and the API key works.
  rpc TestAgentConnection(TestAgentConnectionRequest) returns (TestAgentConnectionResponse) {}
}
```

### 4.4 `platform.proto` — Data source configuration

```protobuf
enum PlatformType {
  PLATFORM_TYPE_UNSPECIFIED = 0;
  PLATFORM_TYPE_DISCORD = 1;
  PLATFORM_TYPE_REDDIT = 2;
  PLATFORM_TYPE_TWITTER = 3;
  PLATFORM_TYPE_LINKEDIN = 4;
  PLATFORM_TYPE_GITHUB = 5;
}

// A platform is a data source from which insights are ingested.
message Platform {
  string id = 1;
  string name = 2;
  PlatformType type = 3;
  // Exactly one config field is set, matching the type.
  oneof config {
    DiscordPlatformConfig discord = 4;
    RedditPlatformConfig reddit = 5;
    TwitterPlatformConfig twitter = 6;
    LinkedInPlatformConfig linkedin = 7;
    GitHubPlatformConfig github = 8;
  }
  google.protobuf.Timestamp created_at = 9;
  google.protobuf.Timestamp updated_at = 10;
}

message DiscordPlatformConfig {
  string bot_token = 1;
  repeated string guild_ids = 2;
  repeated string channel_ids = 3;
}

message RedditPlatformConfig {
  string client_id = 1;
  string client_secret = 2;
  repeated string subreddits = 3;
}

message TwitterPlatformConfig {
  string bearer_token = 1;
  repeated string keywords = 2;
  repeated string accounts = 3;
}

message LinkedInPlatformConfig {
  string access_token = 1;
  repeated string company_ids = 2;
}

message GitHubPlatformConfig {
  string token = 1;
  repeated string repositories = 2;
  bool include_issues = 3;
  bool include_discussions = 4;
}

service PlatformService {
  rpc CreatePlatform(CreatePlatformRequest) returns (CreatePlatformResponse) {}
  rpc GetPlatform(GetPlatformRequest) returns (GetPlatformResponse) {}
  rpc ListPlatforms(ListPlatformsRequest) returns (ListPlatformsResponse) {}
  rpc UpdatePlatform(UpdatePlatformRequest) returns (UpdatePlatformResponse) {}
  rpc DeletePlatform(DeletePlatformRequest) returns (DeletePlatformResponse) {}
  // Validates credentials and connectivity.
  rpc TestPlatformConnection(TestPlatformConnectionRequest) returns (TestPlatformConnectionResponse) {}
}
```

### 4.5 `insight.proto` — Core insight model

```protobuf
// Classification states for insights.
// RAW is the entry state. Classification resolves to exactly one of COLD, HOT, or BURN.
enum InsightState {
  INSIGHT_STATE_UNSPECIFIED = 0;
  INSIGHT_STATE_RAW = 1;    // unprocessed, awaiting classification
  INSIGHT_STATE_COLD = 2;   // uninteresting, irrelevant, general noise
  INSIGHT_STATE_HOT = 3;    // clear sentiment on the product, quantifiable
  INSIGHT_STATE_BURN = 4;   // bugs, incidents, red alerts, product-critical
}

// The medium through which the insight was captured.
enum InsightMedium {
  INSIGHT_MEDIUM_UNSPECIFIED = 0;
  INSIGHT_MEDIUM_POST = 1;
  INSIGHT_MEDIUM_COMMENT = 2;
  INSIGHT_MEDIUM_MESSAGE = 3;
  INSIGHT_MEDIUM_THREAD = 4;
  INSIGHT_MEDIUM_REVIEW = 5;
  INSIGHT_MEDIUM_ISSUE = 6;
  INSIGHT_MEDIUM_DISCUSSION = 7;
}

// An insight is a single unit of social content ingested from a platform.
message Insight {
  string id = 1;
  string pipeline_id = 2;
  string platform_id = 3;
  InsightState state = 4;
  InsightMedium medium = 5;
  string content = 6;                            // raw content text
  string author = 7;                              // platform username/handle
  string source_url = 8;                          // link to original
  string source_id = 9;                           // platform-native ID
  string conversation_id = 10;                    // thread/conversation grouping
  string classification_reason = 11;              // agent's reasoning
  string classifying_agent_id = 12;               // which agent classified
  google.protobuf.Timestamp source_timestamp = 13;
  google.protobuf.Timestamp ingested_at = 14;
  google.protobuf.Timestamp classified_at = 15;
}

service InsightService {
  rpc GetInsight(GetInsightRequest) returns (GetInsightResponse) {}
  rpc ListInsights(ListInsightsRequest) returns (ListInsightsResponse) {}
  // Manual classification override by a human operator.
  rpc ClassifyInsight(ClassifyInsightRequest) returns (ClassifyInsightResponse) {}
  // Aggregate stats: counts by state, platform, time range.
  rpc GetInsightStats(GetInsightStatsRequest) returns (GetInsightStatsResponse) {}
}
```

### 4.6 `pipeline.proto` — Pipeline composition and lifecycle

```protobuf
enum PipelineStatus {
  PIPELINE_STATUS_UNSPECIFIED = 0;
  PIPELINE_STATUS_IDLE = 1;
  PIPELINE_STATUS_RUNNING = 2;
  PIPELINE_STATUS_PAUSED = 3;
  PIPELINE_STATUS_ERROR = 4;
}

// How insights are batched before being sent to an agent for classification.
enum BatchMode {
  BATCH_MODE_UNSPECIFIED = 0;
  BATCH_MODE_SINGLE = 1;            // one insight at a time
  BATCH_MODE_CONVERSATION = 2;      // grouped by conversation/thread
  BATCH_MODE_AUTHOR = 3;            // grouped by author
  BATCH_MODE_TIME_WINDOW = 4;       // grouped by time window
  BATCH_MODE_FIXED_SIZE = 5;        // fixed count per batch
}

message ClassificationStrategy {
  BatchMode batch_mode = 1;
  int32 batch_size = 2;              // for FIXED_SIZE mode
  int32 time_window_seconds = 3;     // for TIME_WINDOW mode
}

// A pipeline connects exactly one product to one or more agents, platforms, and reporters.
message Pipeline {
  string id = 1;
  string name = 2;
  string description = 3;
  string product_id = 4;
  repeated string agent_ids = 5;
  repeated string platform_ids = 6;
  repeated string reporter_ids = 7;
  PipelineStatus status = 8;
  ClassificationStrategy classification_strategy = 9;
  google.protobuf.Timestamp created_at = 10;
  google.protobuf.Timestamp updated_at = 11;
}

service PipelineService {
  rpc CreatePipeline(CreatePipelineRequest) returns (CreatePipelineResponse) {}
  rpc GetPipeline(GetPipelineRequest) returns (GetPipelineResponse) {}
  rpc ListPipelines(ListPipelinesRequest) returns (ListPipelinesResponse) {}
  rpc UpdatePipeline(UpdatePipelineRequest) returns (UpdatePipelineResponse) {}
  rpc DeletePipeline(DeletePipelineRequest) returns (DeletePipelineResponse) {}
  rpc StartPipeline(StartPipelineRequest) returns (StartPipelineResponse) {}
  rpc StopPipeline(StopPipelineRequest) returns (StopPipelineResponse) {}
  rpc PausePipeline(PausePipelineRequest) returns (PausePipelineResponse) {}
  rpc GetPipelineStatus(GetPipelineStatusRequest) returns (GetPipelineStatusResponse) {}
}
```

### 4.7 `reporter.proto` — Output and notification

```protobuf
enum ReporterType {
  REPORTER_TYPE_UNSPECIFIED = 0;
  REPORTER_TYPE_INTERNAL = 1;       // built-in soft reporter (immutable, auto-attached)
  REPORTER_TYPE_WEBHOOK = 2;
  REPORTER_TYPE_EMAIL = 3;
  REPORTER_TYPE_DISCORD = 4;
  REPORTER_TYPE_GITHUB_ISSUE = 5;
}

// Determines which classified insights reach this reporter.
message ReporterFilter {
  repeated InsightState states = 1;       // empty = all non-RAW
  repeated string platform_ids = 2;       // empty = all platforms
}

message Reporter {
  string id = 1;
  string name = 2;
  ReporterType type = 3;
  ReporterFilter filter = 4;
  oneof config {
    WebhookReporterConfig webhook = 5;
    EmailReporterConfig email = 6;
    DiscordReporterConfig discord = 7;
    GitHubIssueReporterConfig github_issue = 8;
  }
  google.protobuf.Timestamp created_at = 9;
  google.protobuf.Timestamp updated_at = 10;
}

message WebhookReporterConfig {
  string url = 1;
  string secret = 2;
}

message EmailReporterConfig {
  string smtp_host = 1;
  int32 smtp_port = 2;
  string from_address = 3;
  repeated string to_addresses = 4;
  string username = 5;
  string password = 6;
}

message DiscordReporterConfig {
  string webhook_url = 1;
}

message GitHubIssueReporterConfig {
  string token = 1;
  string owner = 2;
  string repo = 3;
  repeated string labels = 4;
}

service ReporterService {
  rpc CreateReporter(CreateReporterRequest) returns (CreateReporterResponse) {}
  rpc GetReporter(GetReporterRequest) returns (GetReporterResponse) {}
  rpc ListReporters(ListReportersRequest) returns (ListReportersResponse) {}
  rpc UpdateReporter(UpdateReporterRequest) returns (UpdateReporterResponse) {}
  rpc DeleteReporter(DeleteReporterRequest) returns (DeleteReporterResponse) {}
  // Sends a test payload through the reporter.
  rpc TestReporter(TestReporterRequest) returns (TestReporterResponse) {}
}
```

### 4.8 `report.proto` — Internal report storage

```protobuf
// A report is a snapshot generated by the internal soft reporter.
// One report per pipeline per reporting cycle.
message Report {
  string id = 1;
  string pipeline_id = 2;
  string product_id = 3;
  ReportSummary summary = 4;
  repeated ReportEntry entries = 5;
  google.protobuf.Timestamp generated_at = 6;
}

message ReportSummary {
  int32 total_insights = 1;
  int32 hot_count = 2;
  int32 burn_count = 3;
  int32 cold_count = 4;
}

message ReportEntry {
  string insight_id = 1;
  InsightState state = 2;
  string content_preview = 3;        // truncated content
  string classification_reason = 4;
  string platform_id = 5;
  string source_url = 6;
}

service ReportService {
  rpc GetReport(GetReportRequest) returns (GetReportResponse) {}
  rpc ListReports(ListReportsRequest) returns (ListReportsResponse) {}
}
```

### 4.9 `health.proto` — Retained as-is

Existing health proto remains unchanged.

---

## 5. Go Backend

### 5.1 Package layout

```
cmd/
  discordiance/
    main.go                         # entrypoint, server bootstrap, graceful shutdown

internal/
  config/
    config.go                       # deployment config (Viper, env vars, YAML) — DO NOT MODIFY

  db/
    db.go                           # GORM init, SQLite connection, AutoMigrate

  models/
    product.go                      # Product, ProductContext
    agent.go                        # Agent
    platform.go                     # Platform + typed config tables per PlatformType
    pipeline.go                     # Pipeline + join tables for agents/platforms/reporters
    insight.go                      # Insight
    reporter.go                     # Reporter + typed config tables per ReporterType
    report.go                       # Report, ReportEntry

  rpc/
    server.go                       # HTTP mux, all service handler registration
    interceptors.go                 # logging interceptor (existing)
    services/
      health.go                     # HealthService (existing)
      product.go                    # ProductService implementation
      agent.go                      # AgentService implementation
      platform.go                   # PlatformService implementation
      pipeline.go                   # PipelineService implementation
      insight.go                    # InsightService implementation
      reporter.go                   # ReporterService implementation
      report.go                     # ReportService implementation

  engine/
    engine.go                       # top-level engine: manages pipeline runners
    runner.go                       # single pipeline execution loop
    ingestion.go                    # pulls content from platform adapters → RAW insights
    classification.go               # batches RAW insights → agent → state transition
    dispatch.go                     # routes classified insights → reporters

  platform/
    adapter.go                      # PlatformAdapter interface
    registry.go                     # adapter type registry
    discord/
      adapter.go                    # Discord bot/API adapter

  agent/
    client.go                       # OpenAI-compatible HTTP client
    prompt.go                       # classification prompt builder (uses product context)

  reporter/
    handler.go                      # ReporterHandler interface
    registry.go                     # handler type registry
    internal.go                     # built-in soft reporter (writes Report to DB)
    webhook.go                      # webhook reporter

pkg/
  proto/                            # generated proto code (gitignored, never hand-edited)
```

### 5.2 Key interfaces

```go
// platform/adapter.go
// PlatformAdapter ingests content from an external source.
type PlatformAdapter interface {
    // Start begins ingestion. Content is pushed to the channel.
    Start(ctx context.Context, config *models.Platform, out chan<- RawContent) error
    // Stop halts ingestion gracefully.
    Stop() error
    // Type returns the platform type this adapter handles.
    Type() discordiancev1.PlatformType
}

// RawContent is the normalized output of a platform adapter.
type RawContent struct {
    SourceID       string
    ConversationID string
    Content        string
    Author         string
    SourceURL      string
    Medium         discordiancev1.InsightMedium
    Timestamp      time.Time
}
```

```go
// reporter/handler.go
// ReporterHandler delivers classified insights to an external destination.
type ReporterHandler interface {
    // Deliver sends one or more insights to the configured destination.
    Deliver(ctx context.Context, config *models.Reporter, insights []models.Insight) error
    // Type returns the reporter type this handler handles.
    Type() discordiancev1.ReporterType
}
```

```go
// agent/client.go
// Client talks to any OpenAI-compatible chat completions endpoint.
type Client struct { ... }

// Classify sends insight content (with product context) to the LLM
// and returns the classified state and reasoning.
func (c *Client) Classify(ctx context.Context, agent *models.Agent, product *models.Product, insights []models.Insight) ([]ClassificationResult, error)

type ClassificationResult struct {
    InsightID  string
    State      discordiancev1.InsightState
    Reason     string
}
```

### 5.3 Engine lifecycle

```
Engine.Start()
  └─► for each active pipeline:
        PipelineRunner.Start(pipeline)
          ├─► Ingestion goroutine
          │     adapter.Start() → chan RawContent → write Insight(RAW) to DB
          ├─► Classification goroutine
          │     poll RAW insights → batch per strategy → agent.Classify() → update state in DB
          └─► Dispatch goroutine
                poll newly-classified insights → match reporter filters → handler.Deliver()
```

Each stage is a goroutine connected by DB-backed queues (poll with backoff, not in-memory channels between stages). This ensures crash recovery — no insight is lost if the process restarts.

### 5.4 DB models

GORM models mirror proto structure but are independent types. Key design choices:

- **UUIDs for all primary keys.** Generated server-side.
- **Typed config tables.** `platform_discord_configs`, `platform_reddit_configs`, etc. — joined via `platform_id` FK. Same pattern for reporter configs.
- **Pipeline composition** via join tables: `pipeline_agents`, `pipeline_platforms`, `pipeline_reporters`.
- **Insight state transitions** are atomic updates with `classified_at` timestamp.
- **Soft deletes** on all entities (GORM `DeletedAt`).

---

## 6. Angular Frontend

### 6.1 Technology stack

| Concern | Choice | Rationale |
|---|---|---|
| Framework | Angular (latest) | Requirement |
| Components | PrimeNG | Enterprise-grade, rich data tables, charts, theming |
| Charts | PrimeNG Charts (Chart.js) | Bundled with PrimeNG, zero extra deps |
| RPC Client | @connectrpc/connect-web | Proto-first, type-safe, matches backend contract |
| Proto codegen | @bufbuild/protobuf + @connectrpc/es | Generated from same proto source |
| Styling | PrimeNG Aura theme + PrimeFlex | Modern look, responsive grid |

### 6.2 Application structure

```
web/discordiance/
  angular.json
  package.json
  tsconfig.json
  buf.gen.yaml                            # frontend-specific codegen (ES/TS targets)

  src/
    main.ts
    index.html
    styles.scss

    gen/                                   # generated proto TS (gitignored)
      discordiance/v1/

    app/
      app.component.ts
      app.routes.ts
      app.config.ts

      core/
        services/
          transport.service.ts            # connect-web transport singleton
        interceptors/
          error.interceptor.ts            # global RPC error handling

      shared/
        components/
          status-badge/                   # insight state badge (color-coded)
          confirm-dialog/                 # reusable confirmation
          empty-state/                    # empty state illustrations
        pipes/
          time-ago.pipe.ts
          truncate.pipe.ts

      features/
        dashboard/
          dashboard.component.ts          # overview: pipeline statuses, insight counts, recent burns
          widgets/
            pipeline-status-card/
            insight-breakdown-chart/
            recent-burns-list/

        products/
          product-list.component.ts
          product-detail.component.ts     # context management, linked pipelines
          product-form.component.ts

        pipelines/
          pipeline-list.component.ts
          pipeline-detail.component.ts    # live status, controls (start/stop/pause), insight feed
          pipeline-builder.component.ts   # compose: select product, agents, platforms, reporters

        insights/
          insight-list.component.ts       # filterable table: state, platform, medium, date range
          insight-detail.component.ts     # full content, classification reasoning, source link
          insight-analytics.component.ts  # trends over time, state distribution, platform breakdown

        settings/
          agents/
            agent-list.component.ts
            agent-form.component.ts       # endpoint config + test connection button
          platforms/
            platform-list.component.ts
            platform-form.component.ts    # dynamic form per platform type + test button
          reporters/
            reporter-list.component.ts
            reporter-form.component.ts    # dynamic form per reporter type + filter builder + test
```

### 6.3 Proto codegen for frontend

Add to `buf.gen.yaml` (or a separate `web/discordiance/buf.gen.yaml`):

```yaml
version: v2
plugins:
  - remote: buf.build/bufbuild/es
    out: src/gen
    opt: target=ts
  - remote: buf.build/connectrpc/es
    out: src/gen
    opt: target=ts
```

Angular services wrap the generated connect-web clients, injected via Angular DI. No hand-written API types.

### 6.4 Serving strategy

The Go backend serves the Angular build output as static files from a configurable directory. Single binary deployment — no separate frontend server.

```
GET /api/...     → Connect-RPC handlers
GET /*           → Angular SPA (index.html fallback)
```

---

## 7. Build System

### 7.1 Makefile targets

| Target | Action |
|---|---|
| `make dev` | Build + run backend, serve frontend in dev mode |
| `make build` | Compile Go binary + Angular production build |
| `make proto` | Generate Go + TypeScript from protos (via Buf) |
| `make proto-lint` | Lint proto files |
| `make test` | `go test ./...` + `ng test` |
| `make lint` | `go vet` + `ng lint` + `buf lint` |
| `make fmt` | `go fmt` + `buf format` |
| `make clean` | Remove build artifacts |
| `make deps` | Install Go + npm dependencies |

### 7.2 Buf configuration

`buf.gen.yaml` generates for both Go and TypeScript:

```yaml
version: v2
plugins:
  # Go
  - remote: buf.build/protocolbuffers/go
    out: pkg/proto
    opt: paths=source_relative
  - remote: buf.build/connectrpc/go
    out: pkg/proto
    opt: paths=source_relative
  # TypeScript
  - remote: buf.build/bufbuild/es
    out: web/discordiance/src/gen
    opt: target=ts
  - remote: buf.build/connectrpc/es
    out: web/discordiance/src/gen
    opt: target=ts
```

---

## 8. Removals

The following items from the current repo are removed:

| Item | Reason |
|---|---|
| `web/discordiance-old/` | Dead Svelte frontend, replaced by Angular |
| All deleted Svelte files in `web/discordiance/` | Already staged for deletion |

---

## 9. File Manifest

Complete list of every file in the final project (excluding generated code and dependencies):

```
cmd/discordiance/main.go

internal/config/config.go
internal/db/db.go
internal/models/product.go
internal/models/agent.go
internal/models/platform.go
internal/models/pipeline.go
internal/models/insight.go
internal/models/reporter.go
internal/models/report.go
internal/rpc/server.go
internal/rpc/interceptors.go
internal/rpc/services/health.go
internal/rpc/services/product.go
internal/rpc/services/agent.go
internal/rpc/services/platform.go
internal/rpc/services/pipeline.go
internal/rpc/services/insight.go
internal/rpc/services/reporter.go
internal/rpc/services/report.go
internal/engine/engine.go
internal/engine/runner.go
internal/engine/ingestion.go
internal/engine/classification.go
internal/engine/dispatch.go
internal/platform/adapter.go
internal/platform/registry.go
internal/platform/discord/adapter.go
internal/agent/client.go
internal/agent/prompt.go
internal/reporter/handler.go
internal/reporter/registry.go
internal/reporter/internal.go
internal/reporter/webhook.go

proto/discordiance/v1/common.proto
proto/discordiance/v1/health.proto
proto/discordiance/v1/product.proto
proto/discordiance/v1/agent.proto
proto/discordiance/v1/platform.proto
proto/discordiance/v1/pipeline.proto
proto/discordiance/v1/insight.proto
proto/discordiance/v1/reporter.proto
proto/discordiance/v1/report.proto

web/discordiance/angular.json
web/discordiance/package.json
web/discordiance/tsconfig.json
web/discordiance/src/main.ts
web/discordiance/src/index.html
web/discordiance/src/styles.scss
web/discordiance/src/app/app.component.ts
web/discordiance/src/app/app.routes.ts
web/discordiance/src/app/app.config.ts
web/discordiance/src/app/core/services/transport.service.ts
web/discordiance/src/app/core/interceptors/error.interceptor.ts
web/discordiance/src/app/shared/components/status-badge/...
web/discordiance/src/app/shared/components/confirm-dialog/...
web/discordiance/src/app/shared/components/empty-state/...
web/discordiance/src/app/shared/pipes/time-ago.pipe.ts
web/discordiance/src/app/shared/pipes/truncate.pipe.ts
web/discordiance/src/app/features/dashboard/...
web/discordiance/src/app/features/products/...
web/discordiance/src/app/features/pipelines/...
web/discordiance/src/app/features/insights/...
web/discordiance/src/app/features/settings/...

Makefile
buf.yaml
buf.gen.yaml
go.mod
go.sum
.gitignore
README.md
LICENSE
docs/SYSTEM_DESIGN.md
```

---

## 10. Implementation Order

Phases for building this out, each fully functional before moving to the next:

1. **Proto + DB foundation** — Write all protos, generate code, set up GORM with all models, AutoMigrate.
2. **CRUD services** — Implement all RPC services for products, agents, platforms, pipelines, reporters, reports. Wire into server.
3. **Engine core** — Engine, runner, ingestion, classification, dispatch. Start with single-insight classification and the internal soft reporter.
4. **Platform: Discord** — First platform adapter. End-to-end: Discord messages → RAW insights → classified → internal report.
5. **Reporter: Webhook** — First external reporter. Proves the dispatch pipeline works beyond internal storage.
6. **Angular shell** — Scaffold Angular app with PrimeNG. Connect to backend via connect-web. Dashboard + product/pipeline CRUD.
7. **Angular insights** — Insight browser, analytics charts, manual classification UI.
8. **Angular settings** — Agent, platform, reporter management with test-connection flows.
9. **Batch classification** — Implement remaining batch modes (conversation, author, time window, fixed size).
10. **Additional platforms + reporters** — Reddit, Twitter, email, GitHub issues, etc. as needed.
