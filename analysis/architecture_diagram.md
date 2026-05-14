# Architecture Diagram: HyperWapp

```mermaid
graph TD
    subgraph CLI ["CLI (cmd/root.go)"]
        rootCmd["rootCmd.Run"]
        handleResults["handleResults (Result Consumer)"]
    end

    subgraph Input ["Input Handling (input/)"]
        OnlineInput["Online Resolver"]
        OfflineParser["Offline Parser (Producer)"]
        ProxyServer["Proxy Server"]
    end

    subgraph Workers ["Worker Pool"]
        Worker["Concurrent Workers (Consumers)"]
    end

    subgraph Detect ["Detection Engine (detect/)"]
        WappalyzerEngine["WappalyzerEngine.Detect"]
    end

    subgraph Output ["Output Writers (output/)"]
        CLIWriter["CLI Writer"]
        FileWriter["File Writer (JSON, CSV, etc.)"]
    end

    rootCmd --> OnlineInput
    rootCmd --> OfflineParser
    rootCmd --> ProxyServer

    OnlineInput --> Worker
    OfflineParser --> Worker
    ProxyServer --> Worker

    Worker --> WappalyzerEngine
    WappalyzerEngine --> Worker
    Worker --> handleResults

    handleResults --> CLIWriter
    handleResults --> FileWriter
```

## Data Flow
1. **Initialization:** CLI flags are parsed, and the `WappalyzerEngine` is initialized.
2. **Discovery (Offline):** The input directory is scanned to count total targets for progress tracking.
3. **Production:** An input-specific producer (Parser, Online Resolver, or Proxy) generates `OfflineInput` or `Target` objects.
4. **Processing:** A pool of concurrent workers consumes these inputs and invokes the `WappalyzerEngine`.
5. **Collection:** Detections are sent to a `resultCh`.
6. **Reporting:** `handleResults` processes the detections, maps them to Nuclei tags, and dispatches them to active writers.
