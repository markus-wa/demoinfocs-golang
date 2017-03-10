# Demoinfo helper scripts

The scripts in this folder are helpful for profiling during development and so on.

## Scripts

[//]: # (Disable autolinking with <i></i>)
### gcvis<i></i>.sh

For visualizing memory usage and garbage collection.

Requires [gcvis](https://github.com/davecheney/gcvis).

	go get github.com/davecheney/gcvis

### run_*.sh

Run tests/benchmarks and gather profiling information.

`run_coverage.sh` opens the coverage results (html) after completion.

`run_cpuprof.sh` opens `pprof` after completion.

* Use the parameter `show` to open a graph in your browser instead (`-web` flag).

`run_memprof.sh` opens `pprof` with the `-alloc_space` flag after completion.

* Use the parameter `obj` to start `pprof` with the `-alloc_objects` flag instead.
* Use the parameter `show` to open a graph in your browser instead,

Example: `./run_memprof.sh show` or `./run_memprof.sh obj show`

### show_*.sh

Visuazlize existing profiling information from `demoinfocs-golang/test/results/*`.

`show_{cpu,objects,space}prof.sh` require [graphviz](http://www.graphviz.org).

### analyze_*.sh

Starts `go tool pprof` with existing profiling information.
