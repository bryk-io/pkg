# Package `errors`

When dealing with unexpected or undesired behavior on any system (like issues and
exceptions) the more information available, structured and otherwise, the better.
Preserving error structure and context is particularly important since, in general,
string comparisons on error messages are vulnerable to injection and can even cause
security problems. On distributed systems is very useful, and often required, to
preserve these details across service boundaries as well.

The main goals of this package are:

- Provide a simple, extensible and "familiar" implementation that can be easily
  used as a drop-in replacement for the standard "errors" package and popular 3rd
  party libraries.
- Preserve the entire structure of errors across the wire using pluggable codecs.
- Produce portable and PII-safe error reports. These reports can then be sent to
  any 3rd party service or webhook.
- Enable fast, reliable and secure determination of whether a particular cause
  is present (not relying on the presence of a substring in the error message).
- Being easily composable; by making it extensible with additional error annotations
  and supporting special behavior on custom error types.

## Inspiration

This library is mainly inspired on the original <https://github.com/cockroachdb/errors>
package, while adding some specific adjustments. For additional information about the
original package refer to [PR-36987](https://github.com/cockroachdb/cockroach/pull/36987).

## Motivation

Go provides 4 "idiomatic" ways to inspect errors:

1. Reference comparison to global objects. `err == io.EOF`

2. Type assertions to known error types. `err.(*os.PathError)`

3. Predicate provided by library. `os.IsNotExists(err)`

4. String comparison on the result of `err.Error()`

__Method 1__ breaks down when using wrapped errors, or when transferring errors over
the network.

__Method 2__ breaks down if the error object is converted to a different type. When
wire representations are available, the method is generally reliable; however, if
errors are implemented as a chain of causes, care should be taken to perform the test
on all the intermediate levels.

__Method 3__ is generally reliable although the predicates in the standard library
obviously do not know about any additional custom types. Also, the implementation of
the predicate method can be cumbersome if one must test errors from multiple packages
(dependency cycles). This method loses its reliability if the predicate itself
relies on one of the other methods in a way that's unreliable.

__Method 4__ is the most problematic and unreliable.

## Usage

An error leaf is an object that implements the error interface, but does not refer to
another error via `Unwrap()` and/or `Cause()`.

- To create a new error instance use constructor methods `New()` or `Errorf()`. The
  stack trace of the error will point to the line the method is called.
- You can use `Opaque` to capture an error cause but make it invisible to `Unwrap()`
  or `Is()`. This is particularly useful when a new error occurs while handling another
  one, and the original error must be "hidden".

An error wrapper is an object that implements the error interface, and also refers to
another error via `Unwrap()` and/or `Cause()`.

- Wrapper constructors, i.e., `Wrap()` can be applied safely to a `nil` error; the function
  will behave as no-op in this case.

## Custom error types

You can personalize the behavior of your custom error types by providing implementations
for the following methods:

- `Cause() error`
- `Unwrap() error`
- `Is(target error) bool`

## Redactable Details

You can easily generate a redactable message container that supports manually hiding and
disclosing any additional parameters. This is particularly useful to avoid accidentally
dumping sensitive details to logs or error messages.

```go
// Create a redactable message. All arguments are considered sensitive.
secret := SensitiveMessage("my name is %s (or %s)", "bond", "007")

// Printing the message will remove any arguments used to generate the
// message.
fmt.Println(secret)

// You can use the `%+v` formatting verb to manually disclose the provided
// arguments.
fmt.Printf("%+v", secret)

// When using a redactable message to create an error instance, secret details
// will never be printed out; not even when using the `%+v` formatting verb to
// generate a portable stacktrace.
err := New(secret)
fmt.Printf("%+v", err)
```

## Example

Consider the following dummy code consisting of several levels of function
calling; each one wrapping potential errors bubbling up from lower levels.

```go
// Nested chain of function calls.
// sampleA -> wraps
//  sampleB -> wraps
//   sampleC -> wraps
//    sampleD -> wraps
//     sampleE = returns the original error
func sampleA() error { return Wrap(sampleB(), "a") }
func sampleB() error { return Wrap(sampleC(), "b") }
func sampleC() error { return Wrap(sampleD(), "c") }
func sampleD() error { return Wrap(sampleE(), "d") }
func sampleE() error { return New("deep error") }
```

Your function call this call, receives and error and you need to inspect
it and use it productively.

```go
func myAwesomeFunction() {
  err := sampleA()

  // The most basic thing you can do with the error is log/print it.
  fmt.Println(err.Error())
    // This will print:
    // a: b: c: d: deep error

  // You can also use `Unwrap` to go one level down in the error
  // "chain". For example:
  fmt.Println(Unwrap(err).Error())
    // This will print:
    // b: c: d: deep error

  // Or go straight to the root cause, i.e., the deep-most error
  // in the chain.
  fmt.Println(Cause(err).Error())
    // This will print:
    // deep error
}
```

### Stack Traces

Of course there are more interesting details you can get from your
errors. When diagnosing issues, the more details at your disposal
the better. An important tool at your disposal are stack traces.

```go
func myAwesomeFunction() {
  err := sampleA()

  // Using the '%v' format command with your error value will
  // output a trace formatted as in the standard library
  // `runtime/debug.Stack()`
  fmt.Printf("%v", err)
```

The trace produced will be something similar to:

```sh
a: b: c: d: deep error
/home/ben/go/src/bryk-io/pkg/errors/api_test.go:199 (0x1024a8e4b)
  sampleE: func sampleE() error { return New("deep error") }
/home/ben/go/src/bryk-io/pkg/errors/api_test.go:198 (0x1024a8e38)
  sampleD: func sampleD() error { return Wrap(sampleE(), "d") }
/home/ben/go/src/bryk-io/pkg/errors/api_test.go:197 (0x1024a8deb)
  sampleC: func sampleC() error { return Wrap(sampleD(), "c") }
/home/ben/go/src/bryk-io/pkg/errors/api_test.go:196 (0x1024a8d9b)
  sampleB: func sampleB() error { return Wrap(sampleC(), "b") }
/home/ben/go/src/bryk-io/pkg/errors/api_test.go:195 (0x1024a8d4b)
  sampleA: func sampleA() error { return Wrap(sampleB(), "a") }
/home/ben/go/src/bryk-io/pkg/errors/api_test.go:14 (0x1024a6cdb)
  TestSample: err := sampleA()
/opt/homebrew/Cellar/go/1.19.5/libexec/src/testing/testing.go:1446 (0x1023ebe1b)
  tRunner: fn(t)
```

The standard trace, while helpful, is filled with local details. For
example all the paths from my local filesystem. For this reason, this
package also supports the `%+v` format command that will produce a more
portable and friendlier stack trace output.

```go
func myAwesomeFunction() {
  err := sampleA()

  // Using the '%v' format command with your error value will
  // output a more portable trace.
  fmt.Printf("%+v", err)
```

This time, the trace produced will be formatted like the following:

```sh
a: b: c: d: deep error
‹0› GOPATH/src/bryk-io/pkg/errors/api_test.go:199 (0x102190e4b)
  sampleE: func sampleE() error { return New("deep error") }
‹1› GOPATH/src/bryk-io/pkg/errors/api_test.go:198 (0x102190e38)
  sampleD: func sampleD() error { return Wrap(sampleE(), "d") }
‹2› GOPATH/src/bryk-io/pkg/errors/api_test.go:197 (0x102190deb)
  sampleC: func sampleC() error { return Wrap(sampleD(), "c") }
‹3› GOPATH/src/bryk-io/pkg/errors/api_test.go:196 (0x102190d9b)
  sampleB: func sampleB() error { return Wrap(sampleC(), "b") }
‹4› GOPATH/src/bryk-io/pkg/errors/api_test.go:195 (0x102190d4b)
  sampleA: func sampleA() error { return Wrap(sampleB(), "a") }
‹5› GOPATH/src/bryk-io/pkg/errors/api_test.go:14 (0x10218ecdb)
  TestSample: err := sampleA()
‹6› GOROOT/src/testing/testing.go:1446 (0x1020d3e1b)
  tRunner: fn(t)
```

### Additional Information

Stack traces are a great place to start diagnosing issues, but of course,
the more information available to you the better. This package allow you
to annotate your error with additional contextual information such as:
hints, events and tags.

```go
func myAwesomeFunction() {
  err := sampleA()
  
  // First, cast the error as an "Error" instance provided
  // by this package.
  var te *Error
  if As(err, &te) {
    // Hints provide free-form contextual details.
    te.AddHint("hints can provide additional context about an issue")
    te.AddHint("this was just a test")

    // Tags are usually used for: grouping, filtering and statistic
    // analysis in general.
    te.SetTag("env", "testing")
    te.SetTag("user", "rick")
    te.SetTag("paying_customer", true)

    // You can also add relevant events produced along
    // the error chain. This is often useful when diagnosing
    // complex issues involving many components/services.
    te.AddEvent(Event{
      Kind:    "console",
      Message: "additional debugging information",
    })
  }

  // Then, use the "extended" format command `%+v`. This time
  // the output will include not only the portable stack trace
  // but also all the additional contextual information available
  // in the error.
  fmt.Printf("%+v", te)
```

The output produced from this error will contain a lot more details
that, hopefully, will help diagnose and fix the issue.

```sh
a: b: c: d: deep error
‹0› GOPATH/src/bryk-io/pkg/errors/api_test.go:202 (0x1008a909b)
 sampleE: func sampleE() error { return New("deep error") }
‹1› GOPATH/src/bryk-io/pkg/errors/api_test.go:201 (0x1008a9088)
 sampleD: func sampleD() error { return Wrap(sampleE(), "d") }
‹2› GOPATH/src/bryk-io/pkg/errors/api_test.go:200 (0x1008a903b)
 sampleC: func sampleC() error { return Wrap(sampleD(), "c") }
‹3› GOPATH/src/bryk-io/pkg/errors/api_test.go:199 (0x1008a8feb)
 sampleB: func sampleB() error { return Wrap(sampleC(), "b") }
‹4› GOPATH/src/bryk-io/pkg/errors/api_test.go:198 (0x1008a8f9b)
 sampleA: func sampleA() error { return Wrap(sampleB(), "a") }
‹5› GOPATH/src/bryk-io/pkg/errors/api_test.go:14 (0x1008a6deb)
 TestSample: err := sampleA()
‹6› GOROOT/src/testing/testing.go:1446 (0x1007ebe1b)
 tRunner: fn(t)
‹hints›
 - hints can provide additional context about an issue
 - this was just a test
‹tags›
 - env=testing
 - user=rick
 - paying_customer=true
‹events›
 - (console) additional debugging information
```

### Reports

Finally, having all that information displayed in the console is already
helpful, but being able to collect it elsewhere will be even more so. That's
what `Report` allows us to do. `Report` takes in and error instance and a
`Codec` and is responsible of producing a portable representation of the
error's contents. As a reference, this package includes a `CodecJSON`
implementation.

For example, you can produce a JSON report of the above error and submitted
via HTTP to your monitoring system.

```go
func myAwesomeFunction() {
  err := sampleA()
  
  // First, cast the error as an "Error" instance provided
  // by this package.
  var te *Error
  if As(err, &te) {
    // ... add same additional details as before ...
    // omitted for brevity
  }

  // Produce JSON error report
  js, _ := Report(err, CodecJSON(true))
  fmt.Printf("%s", js)
```

The produced report will be:

```json
{
  "error": "a: b: c: d: deep error",
  "events": [
    {
      "kind": "console",
      "message": "additional debugging information",
      "stamp": 1674845182688
    }
  ],
  "hints": [
    "hints can provide additional context about an issue",
    "this was just a test"
  ],
  "stamp": 1674845182688,
  "tags": {
    "env": "testing",
    "paying_customer": true,
    "user": "rick"
  },
  "trace": [
    {
      "file": "GOPATH/src/bryk-io/pkg/errors/api_test.go",
      "line_number": 198,
      "function": "sampleE",
      "package": "go.bryk.io/pkg/errors",
      "source_line": "func sampleE() error { return New(\"deep error\") }",
      "program_counter": 4297461947
    },
    {
      "file": "GOPATH/src/bryk-io/pkg/errors/api_test.go",
      "line_number": 197,
      "function": "sampleD",
      "package": "go.bryk.io/pkg/errors",
      "source_line": "func sampleD() error { return Wrap(sampleE(), \"d\") }",
      "program_counter": 4297461928
    },
    {
      "file": "GOPATH/src/bryk-io/pkg/errors/api_test.go",
      "line_number": 196,
      "function": "sampleC",
      "package": "go.bryk.io/pkg/errors",
      "source_line": "func sampleC() error { return Wrap(sampleD(), \"c\") }",
      "program_counter": 4297461851
    },
    {
      "file": "GOPATH/src/bryk-io/pkg/errors/api_test.go",
      "line_number": 195,
      "function": "sampleB",
      "package": "go.bryk.io/pkg/errors",
      "source_line": "func sampleB() error { return Wrap(sampleC(), \"b\") }",
      "program_counter": 4297461771
    },
    {
      "file": "GOPATH/src/bryk-io/pkg/errors/api_test.go",
      "line_number": 194,
      "function": "sampleA",
      "package": "go.bryk.io/pkg/errors",
      "source_line": "func sampleA() error { return Wrap(sampleB(), \"a\") }",
      "program_counter": 4297461691
    },
    {
      "file": "GOPATH/src/bryk-io/pkg/errors/api_test.go",
      "line_number": 14,
      "function": "TestSample",
      "package": "go.bryk.io/pkg/errors",
      "source_line": "err := sampleA()",
      "program_counter": 4297453035
    },
    {
      "file": "GOROOT/src/testing/testing.go",
      "line_number": 1446,
      "function": "tRunner",
      "package": "testing",
      "source_line": "fn(t)",
      "program_counter": 4296687131
    }
  ]
}
```
