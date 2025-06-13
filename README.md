### gobench

This is a simplified version of [bent](https://github.com/golang/benchmarks/tree/master/cmd/bent). It automates running Go benchmarks in local directories.

Depends on burntsushi/toml.

Build:
```
$ git clone https://github.com/yukatan1701/gobench
$ cd gobench
$ go build
```

Flags for your use:

| Flag | meaning | example |
| --- | --- | --- |
| -v | print commands as they are run | |
| -N x | benchmark/test repeat count | -N 10 |
| -B file | benchmarks file | -B benchmarks-list.toml |
| -C file | configurations file | -C config.toml |
| -T | path to temporary directory | tmp (by default in the current directory) |
| -b list | run benchmarks in comma-separated list <br> (even if normally "disabled" )| -b uuid,gonum_topo |
| -c list | use configurations from comma-separated list <br> (even if normally "disabled") | -c Tip,Go1.9 |

### Benchmark and Configuration files

Just like bent, gobench expects benchmark and configuration files. They have the similar structure.

The following keys are available to set a configuration:
```
[[Configurations]]
  Name = "conf_example"
  Root = "$HOME/go"
  BuildFlags = ["-tags", "safe"]
  GcFlags = "-N -l"
  LdFlags = "-funcalign=32"
  GcEnv = ["GOARM64=v8.2"]
  RunFlags = ["-test.benchtime=10000000x"]
  RunWrapper = ["/usr/bin/taskset", "-c", "0-3"]
  RunEnv = ["GOGC=off"]
  Disabled = false
```
PGO is supported as well:
```
[[Configurations]]
  Name = "pgo_gen"
  Root = "$HOME/go"
  PgoGen = true # generate profiles for configuration 'pgo_gen'
```
```
[[Configurations]]
  Name = "pgo_use"
  Root = "$HOME/go"
  PgoUse = "pgo_gen" # name of configuration that generated profiles
```
A benchmark description has the following structure:
```
[[Benchmarks]]
  Name = "parallel_copy"
  Benchmarks = "BenchmarkParallelCopy"
  Dir = "$HOME/bench_collection" # Path to directory containing benchmarks (*_test.go)
  RunFlags = ["benchmem", "benchtime=10000000x"]
  Disabled = true
```
`Dir` path can be either absolute or relative. To run benchmarks located in the current directory, set `Dir = "."`.

### Example
```
$ cd examples
$ ../gobench -C config.toml -B bench.toml -v -N 10 | tee base.log
```
By default, `gobench` creates a temporary directory in the current directory. You can change this:
```
$ ../gobench -C config.toml -B bench.toml -v -N 10 -T ../tmp
```
Collect profiles:
```
$ ../gobench -C config-pgo-gen.toml -B bench.toml -v -N 5
```
Use PGO:
```
$ ../gobench -C config-pgo-use.toml -B bench.toml -v -N 10 | tee pgo.log
```
Compare results:
```
$ grep "^Bench" base.log > base.stat
$ grep "^Bench" pgo.log > pgo.stat
$ benchstat base.stat pgo.stat
```