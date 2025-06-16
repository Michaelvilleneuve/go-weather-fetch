# Weather Fetch Go

A Go application that fetches weather forecast data from Météo-France in GRIB2 format, processes it using the eccodes library, and filters geographical data points within a specified polygon region.

## Features

- Downloads AROME weather forecast data from Météo-France
- Processes GRIB2 files using the eccodes C library
- Filters data points within a defined geographical polygon
- Concurrent processing of multiple forecast hours
- Statistical analysis of filtered data points

## Prerequisites

### System Requirements

- Go 1.24.3 or later
- eccodes library (ECMWF's library for reading/writing GRIB files)
- pkg-config

### Installing eccodes Library

The eccodes library is required for parsing GRIB files. Installation varies by operating system:

#### macOS

Using Homebrew (recommended):
```bash
brew install eccodes
```

#### Linux

```bash
sudo apt-get update
sudo apt-get install libeccodes-dev pkg-config
```

## Installation

1. Clone the repository:
```bash
git clone https://github.com/Michaelvilleneuve/weather-fetch-go.git
cd weather-fetch-go
```

2. Create the temporary directory:
```bash
mkdir -p tmp
```

3. Install Go dependencies:
```bash
go mod download
```

4. Build the application:
```bash
go build -o weather-fetch-go
```

## Usage

Run the application:
```bash
./weather-fetch-go
```
