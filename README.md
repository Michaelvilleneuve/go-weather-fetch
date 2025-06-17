# Weather Fetch Go

A Go application that fetches weather forecast data from Météo-France in GRIB2 format, processes it using the eccodes library, and filters geographical data points within a specified polygon region.

This data is then exposed as a JSON API and used in https://lesveusdelavall.org to display a 3D map of highly precised rainfall accumulation map. 

<img width="1512" alt="screenshot overall view" src="https://github.com/user-attachments/assets/019adc4f-ce44-4022-a217-b405736a9fd1" />

<img width="1511" alt="screenshot 3D closeup" src="https://github.com/user-attachments/assets/57cd00f9-5a4e-4bb7-b787-48d87e3d0378" />


## Features

- Downloads AROME weather forecast data from Météo-France
- Processes GRIB2 files using the eccodes C library
- Filters data points within a defined geographical polygon corresponding more or less to the Communidad Valenciana (Spain)
- Concurrent processing of multiple forecast hours
- Statistical analysis of filtered data points
- Serves the data as a JSON file

## Prerequisites

### System Requirements

If you want to run this locally, you need to have the following:

- Go 1.24.3 or later
- eccodes library (ECMWF's library for reading/writing GRIB files)
- pkg-config

Or you can just use the docker if you don't want to install the dependencies

### Dockerless install prerequisites

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

2. Install Go dependencies:
```bash
go mod download
```

3. Run the application:
```bash
go run cmd/weather-fetch/main.go
```

### Deployment

This app is deployed to production using Kamal:
```bash
kamal deploy
```

This will deployed to `weather.lesveusdelavall.org` with SSL auto generated
