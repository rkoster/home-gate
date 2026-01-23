# Home Gate

A Go CLI tool to monitor device usage on Fritz!Box routers and enforce parental control policies.

## Features

- Fetch real-time device usage data from Fritz!Box
- Monitor hourly usage or daily activity
- ASCII timeline visualization for activity
- Parental control policy enforcement with daily limits
- Automatic device blocking when limits are exceeded

## Installation

```bash
go build .
```

## Usage

### Basic Usage

```bash
./home-gate monitor -username <username> -password <password>
```

This will monitor all configured devices and display their daily activity.

### Commands

- `monitor`: Monitor device usage (default command)

### Options

- `--username`: Fritz!Box username (required)
- `--password`: Fritz!Box password (required)
- `--mac`: Specific MAC address to monitor (optional, monitors configured devices if not specified)
- `--period`: "hour" for usage data, "day" for activity monitoring (default: "day")
- `--activity-threshold`: Minimum Byte/s to consider active (default: 0)
- `--policy`: Policy string for allowed minutes per day, e.g., "MO-TH90FR120SA-SU180" (optional)
- `--enforce`: Enforce policy by blocking devices that exceed limits and unblocking compliant ones (optional)

### Examples

Monitor specific device for daily activity:
```bash
./home-gate monitor --username admin --password secret --mac 00:11:22:33:44:55
```

Monitor hourly usage:
```bash
./home-gate monitor --username admin --password secret --period hour
```

Enforce policy (weekdays 90 min, weekends 180 min):
```bash
./home-gate monitor --username admin --password secret --policy "MO-FR90SA-SU180" --enforce
```

## Policy Format

Policies define allowed minutes per day ranges:
- Single days: MO90 (Monday 90 min)
- Ranges: MO-TH90 (Monday to Thursday 90 min)
- Multiple: MO-TH90FR120SA-SU180

## Cron Setup for Enforcement

To run every 15 minutes and enforce limits:

```bash
*/15 * * * * /path/to/home-gate monitor --username admin --password secret --policy "MO-FR90SA-SU180" --enforce
```

## Output

For daily monitoring:
```
Device Name activity in last 12 hours:
Active: 45 minutes (3/48 intervals)
Daily total: 120 minutes (8/96 intervals)
Within policy
Timeline: ...|*.*.*...
```

For hourly monitoring:
```
Device Name usage in last hour:
Downstream: 1024000 bytes
Upstream: 512000 bytes
```

## Requirements

- Go 1.19+
- Access to Fritz!Box router API
- Fritz!Box firmware that supports the monitor API

## Testing

Run tests:
```bash
ginkgo ./...
```

Note: Policy tests are included. Fritz!Box client tests require mocking setup.