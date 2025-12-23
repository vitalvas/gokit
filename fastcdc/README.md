# fastcdc

Fast Content-Defined Chunking (FastCDC) implementation for Go with configurable hash algorithms.

## Features

- FastCDC algorithm with gear rolling hash
- Distributed masks for improved deduplication uniformity
- Configurable chunk sizes with preset configurations
- Hash algorithms: SHA256 (default), SHA384, SHA512
- Streaming support via `io.Reader`
- Zero-allocation iteration methods
- Content-shift resilient (~96% chunk reuse after prefix insertion)

## Usage

### Basic Streaming

```go
reader := bytes.NewReader(data)
chunker, err := fastcdc.NewDefaultChunker(reader)
if err != nil {
    return err
}

for {
    chunk, err := chunker.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }

    fmt.Printf("Offset: %d, Length: %d, Hash: %x\n",
        chunk.Offset, chunk.Length, chunk.Hash)
}
```

### Zero-Allocation Iteration

```go
// ForEach - most efficient, reuses single Chunk object
chunker.ForEach(func(chunk *fastcdc.Chunk) error {
    // process chunk (don't retain reference)
    return nil
})

// ForEachHash - hash only, no data copy
chunker.ForEachHash(func(chunk *fastcdc.Chunk) error {
    // chunk.Data is nil, only Hash/Offset/Length set
    return nil
})

// NextInto - reuse provided Chunk
chunk := &fastcdc.Chunk{Data: make([]byte, 0, config.MaxSize)}
for {
    err := chunker.NextInto(chunk)
    if err == io.EOF {
        break
    }
}
```

### In-Memory Chunking

```go
chunks, err := fastcdc.ChunkBytesDefault(data)
if err != nil {
    return err
}

for _, chunk := range chunks {
    fmt.Printf("Offset: %d, Length: %d\n", chunk.Offset, chunk.Length)
}
```

### Preset Configurations

```go
// Small chunks (8KB avg) - best for small files, high dedup ratio
config := fastcdc.SmallChunkConfig()

// Medium chunks (64KB avg) - general purpose, good balance
config := fastcdc.MediumChunkConfig()

// Large chunks (256KB avg) - best for large files, lower overhead
config := fastcdc.LargeChunkConfig()

// Default (1MB avg) - lowest overhead for large files
config := fastcdc.DefaultConfig()
```

### Custom Configuration

```go
config := fastcdc.Config{
    MinSize:       64 * 1024,           // 64 KB
    MaxSize:       512 * 1024,          // 512 KB
    AvgSize:       128 * 1024,          // 128 KB
    Normalization: 2,
    BufSize:       1024 * 1024,
    HashAlgorithm: fastcdc.HashSHA512,
}

chunker, err := fastcdc.NewChunker(reader, config)
```

### Hash Algorithm Selection

```go
// Using convenience constructor
chunker, err := fastcdc.NewChunkerWithHash(reader, fastcdc.HashSHA512)

// Available algorithms
fastcdc.HashSHA256  // 32 bytes
fastcdc.HashSHA384  // 48 bytes
fastcdc.HashSHA512  // 64 bytes
```

### Standalone Hashing

```go
hash := fastcdc.HashData(data, fastcdc.HashSHA256)
hash := fastcdc.HashDataSHA256(data)
hash := fastcdc.HashDataSHA384(data)
hash := fastcdc.HashDataSHA512(data)
```

## Default Configuration

| Parameter     | Value  | Description                    |
|---------------|--------|--------------------------------|
| MinSize       | 512 KB | Minimum chunk size             |
| MaxSize       | 8 MB   | Maximum chunk size             |
| AvgSize       | 1 MB   | Target average chunk size      |
| Normalization | 2      | Chunk size distribution control|
| HashAlgorithm | SHA256 | Hash algorithm                 |

## Chunk Structure

```go
type Chunk struct {
    Offset   uint64   // Byte offset in the original data stream
    Length   uint64   // Length of the chunk in bytes
    Data     []byte   // Chunk data (nil when using NextHash/ForEachHash)
    Hash     [64]byte // Hash of the chunk data (full size)
    HashSize int      // Actual hash size (32 for SHA256, 48 for SHA384, 64 for SHA512)
}
```

## Performance

Benchmarks on Apple M3 Pro (100MB test data):

### By Preset Configuration

| Config       | Avg Size | Throughput | Allocs/op |
|--------------|----------|------------|-----------|
| Small        | 8 KB     | ~1.08 GB/s | 10644     |
| Medium       | 64 KB    | ~1.11 GB/s | 1328      |
| Large        | 256 KB   | ~1.00 GB/s | 338       |
| Default      | 1 MB     | ~1.11 GB/s | 83        |

### By Operation Type

| Operation             | Throughput  | Notes                    |
|-----------------------|-------------|--------------------------|
| Pure Boundary Finding | ~2.2 GB/s   | No hashing overhead      |
| With SHA256 Hash      | ~1.1 GB/s   | Full chunking pipeline   |
| SHA256 Standalone     | ~3.0 GB/s   | Hash computation only    |
