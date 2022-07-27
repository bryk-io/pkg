/*
Package errors provides an enhanced error management library.

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
original package refer to [PR-36987](https://github.com/cockroachdb/cockroach/pull/36987)
*/
package errors
