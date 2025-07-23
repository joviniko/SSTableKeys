# SSTableKeys

A Go tool for analyzing Cassandra SSTable index files. SSTableKeys processes IDX0 files to extract and count network-related data including protocols, ports, IPv4/IPv6 addresses, and calculates total data sizes from corresponding PKT0 files.

## Features

- Concurrent processing with configurable worker pools (default: 4 workers)
- Extracts multiple data types based on key prefixes:
  - Type 1: Protocol data
  - Type 2: Port data  
  - Type 4: IPv4 addresses
  - Type 6: IPv6 addresses
- Thread-safe counting using mutexes
- Optional timestamp filtering for processing data within specific time ranges
- JSON output format

## Dependencies

- `github.com/golang/leveldb/table` - For reading SSTable files
- Standard Go libraries for networking, JSON, synchronization

## Installation

### Build from source

```bash
go build SSTableKeys.go
```

### Build optimized binary

```bash
go build -ldflags="-s -w" SSTableKeys.go
```

## Usage

### Basic usage

Process all files in IDX0 directory:
```bash
./SSTableKeys /path/to/IDX0/
```

### Timestamp filtering

Process files within timestamp range (Unix timestamps):
```bash
./SSTableKeys /path/to/IDX0/ 1234567890 1234567900
```

## Development

### Format code
```bash
go fmt SSTableKeys.go
```

### Run with race detection
```bash
go run -race SSTableKeys.go /path/to/data/
```

## How it works

1. **File Processing**: Only processes files matching regex `^\d{16}$`
2. **Version Check**: Ensures major version compatibility (version 2)
3. **Concurrent Processing**: Uses worker pools with WaitGroups and channels
4. **Data Extraction**: Reads SSTable index files using LevelDB's table reader
5. **Path Mapping**: Derives data folder path by replacing "IDX0" with "PKT0"
6. **Output**: Aggregates results and outputs as JSON

## Architecture

The application is built as a single-file Go program that implements:
- LevelDB table reader for parsing SSTable files
- Concurrent processing with configurable worker pools
- Thread-safe data counting with mutexes for each data type
- Optional timestamp-based file filtering

## License

See LICENSE file for details.