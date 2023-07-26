import opentelemetry from '@opentelemetry/api';

// API is required to instrument an application manually.
// https://opentelemetry.io/docs/instrumentation/js/manual

const tracerName = '@bryk-io/otel';
const tracerVersion = '0.1.0';

/**
 * Start a new span with the given name and execute the provided function.
 * The span will be automatically closed once the function returns.
 * @param {string} name Span name.
 * @returns {object}
 */
export function Start(name) {
  return opentelemetry.trace.getTracer(tracerName, tracerVersion).startSpan(name);
}

/**
 * Retrieve the currently active span.
 * @returns {object}
 */
export function GetActiveSpan() {
  return opentelemetry.trace.getActiveSpan();
}
