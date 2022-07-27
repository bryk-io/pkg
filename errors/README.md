# Summary

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

This library is mainly inspired on the original https://github.com/cockroachdb/errors
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

```
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
