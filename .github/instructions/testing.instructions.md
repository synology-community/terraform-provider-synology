---
applyTo: '**/*_test.go'
---
## Testing conventions

- All new packages and most new significant functionality must come with unit tests.
- Table-driven tests are preferred for testing multiple scenarios/inputs. For an example, see TestNamespaceAuthorization.
- Significant features should come with integration (test/integration) and/or end-to-end (test/e2e) tests.
- Including new kubectl commands and major features of existing commands.
- Unit tests must pass on macOS and Windows platforms - if you use Linux specific features, your test case must either be skipped on windows or compiled out (skipped is better when running Linux specific commands, compiled out is required when your code does not compile on Windows).
- Avoid relying on Docker Hub. Use the Google Cloud Artifact Registry instead.
- Do not expect an asynchronous thing to happen immediately—do not wait for one second and expect a pod to be running. Wait and retry instead.

## Use table-drive tests, and consistently use tt for a test case

Try to use table-driven tests whenever feasible, but it’s okay to just copy some code when it’s not; don’t force it (e.g. sometimes it’s easier to write a table-driven test for all but one or two cases; be practical).

Consistently using the same variable name for a test case will make it easier to work with large code bases. You don’t have to use tt, but it is the most commonly used in Go’s standard library (564 times, vs. 116 for tc).

Also see TableDrivenTests.

### Example:

```go
tests := []struct {
    // ...
}{}

for _, tt := range tests {
}
```

## Use subtests 

Using subtests makes it possible to run just a single test case from a table, as well as easily see which test exactly failed. Even though subtests were added back in 2016, I sometimes still see people not using them for new tests.

I tend to simply use the test number if it’s obvious what is being tested, and add a test name if it’s not or if there are many test cases.

Also see Using Subtests and Sub-benchmarks.

Example:

```go
tests := []struct {
    // ...
}{}

for _, tt := range tests {
    t.Run("", func(t *testing.T) {
        have := TestFunction(tt.input)
        if have != tt.want {
            t.Errorf("failed for %v ...", tt.input)
        }
    })
}
```

## Don’t ignore errors 

I frequently see people ignore errors in tests. This is a bad idea and can make for some confusing test failures.

Example:

```go
have, err := Fun()
if err != nil {
    t.Fatalf("unexpected error: %v", err)
}
```

or:

```go
have, err := Fun()
if err != tt.wantErr {
    t.Fatalf("wrong error\ngot:  %v\nwant: %v", err, tt.wantErr)
}
```

I often use ErrorContains, which is a useful helper function for testing error messages (avoids some if err != nil && [..]).

## Use want and have 

want is shorter than expected, have is shorter than actual. Shorter names is always good, IMHO, and is especially beneficial for aligning output (see next item).

Example:

```go
cases := []struct {
    ...
    want     string
    wantCode int
}{}
```

Add useful, aligned, information 

It’s annoying when a test fail with a useless error message, or a noisy error message which makes it hard to see what exactly went wrong.

This is not especially useful:

```go
t.Errorf("wrong output: %v", have)
When it fails it tells us we have the wrong output, but what did we even expect to have?
```

This is better:

```go
name := "test foo"
want := "this string"
t.Run(name, func(t *testing.T) {
    have := "this string!"
    t.Errorf("wrong output for %v, want %v; have %v", name, have, want)
})
```

With the downside that it’s hard to see what exactly failed:

```
--- FAIL: TestX (0.00s)
    --- FAIL: TestX/test_foo (0.00s)
        a_test.go:9: wrong output for test foo, want this string!; have this string
When aligned, this is a lot easier:

name := "test foo"
want := "this string"
t.Run(name, func(t *testing.T) {
    have := "this string!"
    t.Errorf("wrong output\ngot: %q\nwant: %q", have, want)
})
--- FAIL: TestX (0.00s)
    --- FAIL: TestX/test_foo (0.00s)
        a_test.go:10: wrong output
                have: "this string!"
                want: "this string"
```

This is also who I like want and have: they’re of identical length so it’s easy to align.

I also tend to prefer to use %q or %#v, as that will show things like trailing whitespace or unprintable characters more clearly.