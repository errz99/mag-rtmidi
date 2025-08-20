## go-rtmidi
[![Go Report Card](https://goreportcard.com/badge/github.com/mattrtaylor/go-rtmidi)](https://goreportcard.com/report/github.com/mattrtaylor/go-rtmidi)
![Build](https://github.com/mattrtaylor/go-rtmidi/actions/workflows/ci.yml/badge.svg?event=push)
[![codecov](https://codecov.io/gh/mattrtaylor/go-rtmidi/branch/main/graph/badge.svg?token=KKYZ8CRVTU)](https://codecov.io/gh/mattrtaylor/go-rtmidi)


This is a copy of [RtMidi](https://www.music.mcgill.ca/~gary/rtmidi/)'s [Go library](https://github.com/thestk/rtmidi/tree/master/contrib/go/rtmidi)
with a repository structure that allows it to be imported.

```
$ go get github.com/mattrtaylor/go-rtmidi
```



#### Benchmarks
```
$ go test -bench=. -benchtime=5s
goos: darwin
goarch: amd64
pkg: github.com/mattrtaylor/go-rtmidi
cpu: Intel(R) Core(TM) i7-8850H CPU @ 2.60GHz
BenchmarkNoteOn-12           8343454     785.0 ns/op       3.82 MB/s
BenchmarkNotes24-12         10487269     612.5 ns/op     235.09 MB/s
BenchmarkNotes96-12          9802315     628.8 ns/op     916.07 MB/s
BenchmarkNotes256-12        10772338     614.6 ns/op    2499.21 MB/s
BenchmarkNotes1024-12        9502921     620.9 ns/op    9894.64 MB/s
BenchmarkSysEx7-12           8790547     753.7 ns/op       9.29 MB/s
BenchmarkSysEx60-12          8528662     742.6 ns/op      80.80 MB/s
BenchmarkSysEx512-12         7865480     809.5 ns/op     632.52 MB/s
BenchmarkSysEx1024-12        7242057     810.3 ns/op    1263.72 MB/s
BenchmarkSysEx2048-12        7076628     854.4 ns/op    2396.87 MB/s
BenchmarkSysEx4096-12        1504501      4940 ns/op     829.18 MB/s
BenchmarkSysEx65535-12         12180    463423 ns/op     141.42 MB/s
PASS
ok      github.com/mattrtaylor/go-rtmidi    96.722s
```
