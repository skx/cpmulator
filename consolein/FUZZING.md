# Fuzz Testing

Fuzz-testing involves creating random input, and running the program with it, to see what happens.

The expectation is that most of the random inputs will be invalid, so you'll be able to test your error-handling and see where you failed to consider things appropriately.



## Usage

Within this directory just run:

```
$ go test -fuzztime=300s -parallel=1 -fuzz=FuzzOptionsParser -v
```

You'll see output like so, reporting on progress:

```
..
=== RUN   FuzzOptionsParser
fuzz: elapsed: 0s, gathering baseline coverage: 0/106 completed
fuzz: elapsed: 2s, gathering baseline coverage: 106/106 completed, now fuzzing with 1 workers
fuzz: elapsed: 3s, execs: 145 (48/sec), new interesting: 0 (total: 106)
fuzz: elapsed: 6s, execs: 359 (71/sec), new interesting: 0 (total: 106)
fuzz: elapsed: 9s, execs: 359 (0/sec), new interesting: 0 (total: 106)
fuzz: elapsed: 12s, execs: 359 (0/sec), new interesting: 0 (total: 106)
fuzz: elapsed: 15s, execs: 359 (0/sec), new interesting: 0 (total: 106)
```

If you have a beefier host you can drop the `-parallel=1` flag, to run more jobs in parallel, and if you wish you can drop the `-fuzztime=300s` flag to continue running fuzz-testing indefinitely (or until an issue has been found).
