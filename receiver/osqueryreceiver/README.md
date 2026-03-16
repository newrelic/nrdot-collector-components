# OS Query Receiver

An OTel receiver for [osquery](https://osquery.readthedocs.io/en/stable/).

## Sample Config

Put below yaml block in your OTel receiver config. You can run queries according to your usecase.

```yaml
osqueryreceiver:
  tmp_dir: /tmp/osqueryreceiver/tmp_data/
  extensions_socket: /var/osquery/osquery.em
  interval: 60s
  collections:
    - system_info
    - package_info
    # - processes
    # - users
  custom_queries:
    # - select * from processes limit 5;
    # - select * from os_version;
    # - select * from logged_in_users;
    # - select * from listening_ports limit 5;
    # - select * from packages limit 5;
```

## Diagrams

### Overview

```mermaid
%%{init: {'theme': 'light'}}%%
graph TD
    subgraph Host
        CFG[config.yaml] --> RCV[OSQueryReceiver]
        RCV --> MGR[OSQueryManager]
        MGR --> STMGR[In-memory/File State Manager]
        MGR --> CACHE[Cache Manager]
        MGR --> EXEC[Collection Executor]
        EXEC --> COLLS[Collections]
        COLLS -->|SQL Queries| OSQ[osqueryi CLI]
        OSQ -->|JSON Results| EXEC
        EXEC --> STMGR
        STMGR -->|Changed Rows| MGR
        CACHE --> MGR
    end

    MGR --> PLOG[OTel Logs Pipeline]
    PLOG --> EXPORT[Configured Exporters]
```

### Sequence Diagram

```mermaid
sequenceDiagram
    participant Receiver as OSQuery Receiver
    participant Manager as OSQuery Manager
    participant Cache as Cache Manager
    participant Executor as Collection Executor
    participant Collection as ICollection
    participant OSQuery as OSQuery CLI
    participant Consumer as Log Consumer

    Note over Receiver,Manager: Initialization Phase
    Receiver->>Manager: NewOSQueryManager(config, logger)
    activate Manager
    Manager->>Manager: RegisterCollections(config)
    
    loop For each collection in config.Collections
        Manager->>Collection: GetCollection(collectionName)
        Collection-->>Manager: ICollection instance
        Manager->>Executor: Append to Collections[]
    end
    
    loop For each custom query
        Manager->>Collection: GetCustomCollection(name, query)
        Collection-->>Manager: Custom ICollection
        Manager->>Executor: Append to Collections[]
    end
    
    Manager-->>Receiver: OSQueryManager instance
    deactivate Manager
    
    Note over Receiver,Consumer: Collection Phase (Periodic)
    
    Receiver->>Manager: collect(nextConsumer)
    activate Manager
    
    Manager->>Executor: ExecuteAll()
    activate Executor
    
    par Execute Collections in Parallel
        loop For each collection in Collections[]
            Executor->>Collection: GetQuery()
            Collection-->>Executor: SQL query string
            
            Executor->>Executor: Run(query)
            activate Executor
            Executor->>OSQuery: osqueryi --json <query>
            OSQuery-->>Executor: JSON output
            Executor->>Executor: json.Unmarshal(output)
            deactivate Executor
            
            Executor->>Collection: Unmarshal(data)
            Collection->>Collection: Transform to structured type
            Collection-->>Executor: Structured result
            
            Note over Executor: QueryExecution{<br/>Query, TransformInto,<br/>ExecutedAt, Error}
        end
    end
    
    Executor-->>Manager: map[string]QueryExecution
    deactivate Executor
    
    Manager->>Cache: UpdateCache(collectionName, data)
    Cache-->>Manager: Cached
    
    Manager->>Manager: sendToConsumer(results)
    activate Manager
    
    loop For each QueryExecution result
        Manager->>Manager: Convert to plog.Logs
        Note over Manager: Add query, collection,<br/>timestamp attributes
        Manager->>Manager: Marshal TransformInto to JSON
        Manager->>Consumer: ConsumeLogs(logs)
        Consumer-->>Manager: Success/Error
    end
    
    deactivate Manager
    deactivate Manager
```

### Class Diagram

```mermaid
classDiagram
    class ICollection {
        <<interface>>
        +GetName() string
        +GetQuery() string
        +Unmarshal(any) interface
    }

    class SystemInfoCollection {
        +Hostname string
        +UUID string
        +CPUType string
        +CPUSubtype string
        +CPUBrand string
        +PhysicalMemory string
        +HardwareVendor string
        +HardwareModel string
        +GetName() string
        +GetQuery() string
        +Unmarshal(any) interface
    }

    class CustomCollection {
        +name string
        +query string
        +GetName() string
        +GetQuery() string
        +Unmarshal(any) interface
    }

    class QueryExecution {
        +Query string
        +TransformInto interface
        +ExecutedAt time.Time
        +Error error
    }

    class CollectionExecutor {
        +logger *zap.Logger
        +Collections []ICollection
        +ExecuteAll() map[string]QueryExecution
        +Run(query string) (any, error)
    }

    class OSQueryManager {
        +extensionsSocket string
        +logger *zap.Logger
        +executor *CollectionExecutor
        +cache CacheManager
        +RegisterCollections(config) error
        +collect(nextConsumer) error
        -sendToConsumer(ctx, results, consumer) error
    }

    class CacheManager {
        +cache map[string]CachedResult
        +cacheMutex sync.RWMutex
        +logger *zap.Logger
        +UpdateCache(collectionName, data)
        +GetCachedResult(collectionName) (interface, bool)
        +InvalidateCache(collectionName)
    }

    class CachedResult {
        +Data interface
        +CachedAt time.Time
        +TTL time.Duration
        +IsValid bool
    }

    class Config {
        +CollectionInterval string
        +ExtensionsSocket string
        +Collections []string
        +CustomQueries []string
        +Validate() error
    }

    class OSQueryReceiver {
        +host component.Host
        +cancel context.CancelFunc
        +logger *zap.Logger
        +nextConsumer consumer.Logs
        +config *Config
        +Start(ctx, host) error
        +Shutdown(ctx) error
    }

    class CollectionFactory {
        <<utility>>
        +GetCollection(name) (ICollection, error)
        +GetCustomCollection(name, query) ICollection
    }

    ICollection <|.. SystemInfoCollection
    ICollection <|.. CustomCollection
    
    CollectionExecutor o-- ICollection : contains multiple
    CollectionExecutor --> QueryExecution : produces
    
    OSQueryManager --> CollectionExecutor : uses
    OSQueryManager --> CacheManager : uses
    OSQueryManager --> Config : reads
    
    CacheManager o-- CachedResult : stores
    
    OSQueryReceiver --> OSQueryManager : creates and uses
    OSQueryReceiver --> Config : uses
    
    CollectionFactory ..> ICollection : creates
    OSQueryManager ..> CollectionFactory : uses

    note for ICollection "All collections implement this interface.<br/>GetQuery() returns SQL, Unmarshal()<br/>transforms raw data to typed structs"
    note for CollectionExecutor "Executes all collections in parallel<br/>using sync.WaitGroup"
    note for OSQueryManager "Central orchestrator that manages<br/>collection lifecycle and result processing"
```

## TODO

* What format do we send the data in? Right now the idea is to use logs. But we need to define it.
* What would be the communication medium for Inventory manager with the backend service? How would it instruct the collector to resend entire inventory in case of data corruption/loss?
