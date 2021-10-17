# Collect
[![Go Report Card](https://goreportcard.com/badge/github.com/tommsawyer/collect)](https://goreportcard.com/report/github.com/tommsawyer/collect)
[![codecov](https://codecov.io/gh/tommsawyer/collect/branch/main/graph/badge.svg?token=63GFZ0O3OR)](https://codecov.io/gh/tommsawyer/collect)

Allows you to collect all pprof profiles with one command.

## Installation
Just go-get it:
```bash
$ go get github.com/tommsawyer/collect/cmd/collect
```

## Motivation

Sometimes I need to quickly collect all pprof profiles for future optimization. Its very frustrating to do it with long curl commands like:
```bash
$ curl -sK -v http://localhost:8080/debug/pprof/heap > heap.out && curl -sK -v http://localhost:8080/debug/pprof/allocs > allocs.out && curl -sK -v http://localhost:8080/debug/pprof/goroutine > goroutine.out && curl -sK -v http://localhost:8080/debug/pprof/profile > profile.out && curl -o ./trace "http://localhost:8080/debug/pprof/trace?debug=1&seconds=20"

```

Also it:
- doesnt run concurrently, resulting in slow execution
- you have to manually move profiles to some directories if you want to store them for future comparsion
- you need to wait for the command to complete and run it again if you want to collect profiles several times

## Usage
Provide url from which profiles will be scraped:
```bash
$ collect -u=http://localhost:8080
```
This will download allocs, heap, goroutine and cpu profiles and save it into directory structure like this:

```
- localhost 8080
  - YYYY MM DD
    - HH MM SS
      - allocs
      - heap
      - profile
      - goroutine
```

You can provide as many urls as you want:
```bash
$ collect -u=http://localhost:8080 -u=http://localhost:7070
```

You can choose which profiles will be scraped:
```bash
$ collect -p=allocs -p=heap -u=http://localhost:8080
```

Query parameters for profiles are also supported:
```bash
$ collect -p=trace\?seconds\=20 -u=http://localhost:8080
```

Use `-l` flag to collect profiles in endless loop(until Ctrl-C). This will collect profiles each 60 seconds (you can redefine interval with `-i`).
```bash
$ collect -l -u=http://localhost:8080
```

## Command-Line flags
| Flag        | Default                        | Usage                                     |
| ----------- | -------------------------------| ----------------------------------------- |
| -u          |                                | url from which profiles will be collected.|
| -p          | allocs,heap,goroutine,profile  | profiles to collect.                      |
| -l          | false                          | collect profiles in endless loop          |
| -i          | 60s                            | interval between collecting. use with -l  |
