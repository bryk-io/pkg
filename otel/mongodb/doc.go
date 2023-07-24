/*
Package mongodb provides OTEL instrumentation for the MongoDB Go Driver.

This package simply provides a wrapper around the original Go-contrib version
of the MongoDB instrumentation, so that it can be imported as a single dependency
and avoid issues with multiple versions of the same instrumentation.
*/
package mongodb
