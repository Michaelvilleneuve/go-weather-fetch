# Weather Fetch Go

A Go application that fetches weather forecast data from Météo-France in GRIB2 format, processes it using the eccodes library, and filters geographical data points within a specified polygon region.

This data is then exposed as a JSON API and used in https://lesveusdelavall.org to display a 3D map of highly precised rainfall accumulation map. 


![Screenshot 2025-06-19 at 22 34 47](https://github.com/user-attachments/assets/e2ddf00b-1071-4555-a444-3ab8db2189fe)
![Screenshot 2025-06-19 at 22 34 19](https://github.com/user-attachments/assets/ed945814-085a-4bc3-a5e0-7414abecb8b4)
![Screenshot 2025-06-19 at 22 33 39](https://github.com/user-attachments/assets/a14c2c9e-6f71-4251-8d3e-9ad1a6eb3503)
![Screenshot 2025-06-19 at 22 32 46](https://github.com/user-attachments/assets/81f8d537-0834-411c-b468-4a28e9039de0)
![Screenshot 2025-06-17 at 15 12 24](https://github.com/user-attachments/assets/6f170f5c-c085-4f75-9cf1-cd67f075c069)
![Screenshot 2025-06-17 at 15 11 56](https://github.com/user-attachments/assets/30cbba20-4ef0-4c20-9047-8d4f343534e0)


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
