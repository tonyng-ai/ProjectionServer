# ProjectionServer Diagrams

This document captures the core ProjectionServer workflows and runtimes inferred from the Go service that synchronizes Microsoft SQL Server data into PostgreSQL and exposes projection-ready views over a Gin HTTP API.

## Data Flow Overview

```mermaid
flowchart LR
    Operator([Ops / Analyst]) -->|requests dashboards| Frontend[React Projection UI]
    Frontend -->|HTTP /api/*| APIServer[Gin API Server]
    Config[(sync-config.yaml)] --> APIServer
    APIServer -->|control messages| Coordinator[CoordinatorActor]
    Coordinator -->|spawn/dispatch| SyncActors[SyncActor per Table]
    SyncActors -->|invoke| SyncEngine[Sync Engine]
    SyncEngine -->|uses| DBMgr[Database Manager]
    DBMgr -->|query| SourceDB[(MSSQL Source)]
    SyncEngine -->|writes| TargetDB[(PostgreSQL Target)]
    TargetDB -->|projection queries| APIServer
    APIServer -->|JSON rows| Frontend
```

## Projection Data Request Sequence

```mermaid
sequenceDiagram
    participant UI as React UI
    participant API as APIHandler (Gin)
    participant Cfg as Config (ProjectionConfig)
    participant DB as DatabaseManager.Target

    UI->>API: GET /api/projections/:id/data
    API->>Cfg: Lookup projection metadata
    alt Projection exists
        API->>DB: Query target view with filters & sort
        DB-->>API: Row set + aggregations
        API-->>UI: 200 OK { rows, totals, meta }
    else Projection missing
        API-->>UI: 404 Projection not found
    end
```

## Sync Actor Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Idle
    Idle --> Scheduled : ProtoActor trigger enabled
    Scheduled --> Syncing : ScheduleSyncMessage
    Syncing --> Reporting : Send SyncResultMessage to Coordinator
    Reporting --> Scheduled : Schedule next sync (auto)
    Reporting --> Idle : ProtoActor trigger disabled
    Syncing --> Idle : Stop/Shutdown
    Syncing --> Idle : Error encountered
    Scheduled --> Idle : Stop/Shutdown
```

These diagrams emphasize how projection requests traverse the service, how asynchronous synchronization feeds the projection data sets, and how configuration drives each component.
