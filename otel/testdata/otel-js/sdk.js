import { ZoneContextManager } from '@opentelemetry/context-zone';
import {
  CompositePropagator,
  W3CBaggagePropagator,
  W3CTraceContextPropagator
} from '@opentelemetry/core';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { registerInstrumentations } from '@opentelemetry/instrumentation';
import { DocumentLoadInstrumentation } from '@opentelemetry/instrumentation-document-load';
import { FetchInstrumentation } from '@opentelemetry/instrumentation-fetch';
import { XMLHttpRequestInstrumentation } from '@opentelemetry/instrumentation-xml-http-request';
import { Resource, browserDetector, detectResourcesSync } from '@opentelemetry/resources';
import { BatchSpanProcessor, WebTracerProvider } from '@opentelemetry/sdk-trace-web';

// Sentry/OpenTelemetry integration
// https://www.npmjs.com/package/@sentry/opentelemetry-node
// https://github.com/getsentry/opentelemetry-demo
import { SentryPropagator, SentrySpanProcessor } from '@sentry/opentelemetry-node';
import * as Sentry from '@sentry/svelte';

/**
 * @typedef {Object} SentryOptions
 * @property {boolean} [enabled=false] Whether to activate the Sentry tracing integration.
 * @property {string} dsn The DSN used to connect to Sentry and identify the project.
 *   If omitted, the SDK will not send any data to Sentry.
 * @property {string} release The release identifier used when uploading respective
 *   source maps. Specify this value to allow Sentry to resolve the correct source maps when processing events.
 * @property {number} [tracesSampleRate=1.0] Sample rate to determine trace sampling.
 *   0.0 = 0% chance of a given trace being sent (send no traces)
 *   1.0 = 100% chance of a given trace being sent (send all traces)
 * @property {boolean} [enableSessionReplay=false] Whether to enable automatic Session Tracking.
 *   https://docs.sentry.io/platforms/javascript/session-replay/
 * @property {number} [replaysSessionSampleRate=0.5] The sample rate for session-long replays.
 *   1.0 will record all sessions and 0 will record none.
 * @property {number} [replaysOnErrorSampleRate=1.0] The sample rate for sessions that has had an error occur.
 *  This is independent of `replaysSessionSampleRate`.
 *  1.0 will record all sessions and 0 will record none.
 */

/**
 * @typedef {Object} InstrumentationOptions
 * @property {string} serviceName Value to use as `service.name` for spans.
 * @property {string} serviceVersion Value to use as `service.version` for spans.
 * @property {string} [otlpExporter=''] OTLP exporter endpoint; if provided, spans will be sent to this address
 *   using HTTP/JSON.
 * @property {boolean} [useZoneContext=false] Whether to use the zone context manager to enable tracing action between asynchronous
 *   operations in web. It was not possible with the standard "stack context manager".
 *   It stores the information about context in zone. Each Context will have always new
 *   Zone; it also supports binding a certain Span to a target that has "addEventListener"
 *   and "removeEventListener". When this happens a new zone is being created and the
 *   provided Span is being assigned to this zone.
 *   https://github.com/open-telemetry/opentelemetry-js/tree/main/packages/opentelemetry-context-zone
 * @property {string[]} [propagateTraceHeaderCorsUrls=[]] URLs which should include trace headers when origin doesn't match.
 * @property {SentryOptions} [sentry={}] Sentry integration settings.
 * @see https://opentelemetry.io/docs/js/instrumentation/
 */

/**
 * @typedef {Object} Tracer
 * @property {function} setup Setup the instrumentation provider.
 */

/**
 * Setup Sentry integration.
 * @param {SentryOptions} options Sentry configuration options.
 * @returns {void}
 */
function setupSentry(options) {
  if (!options.enabled) {
    return; // no-op if not enabled
  }

  // custom integrations enabled
  let integrations = [];
  if (options.enableSessionReplay) {
    integrations.push(new Sentry.Replay());
  }

  // initialize Sentry using official SDK
  // ~ right now the svelte package is used by default.
  Sentry.init({
    // project identifiers
    dsn: options.dsn,
    release: options.release,

    // trace 100% of transactions; not recommended for production
    tracesSampleRate: options.tracesSampleRate,
    // set the instrumenter to use OpenTelemetry instead of Sentry
    instrumenter: 'otel',

    // capture Replay for 10% of all sessions,
    // plus for 100% of sessions with an error
    integrations: integrations,
    replaysSessionSampleRate: options.replaysSessionSampleRate,
    replaysOnErrorSampleRate: options.replaysOnErrorSampleRate
  });
}

/**
 * Setup the instrumentation provider. This is not required to instrument an
 * application, but rather to collect Telemetry data once the application is
 * instrumented.
 *
 * @param {InstrumentationOptions} options Instrumentation configuration options.
 * @returns {void}
 * @see https://opentelemetry.io/docs/js/instrumentation/
 */
export function TracerSetup(options) {
  if (typeof window === 'undefined') {
    return; // no-op if not running in a browser
  }

  // configure OTEL resource
  let resource = new Resource({
    'service.name': options.serviceName,
    'service.version': options.serviceVersion
  }).merge(detectResourcesSync({ detectors: [browserDetector] }));

  // create browser tracer
  const provider = new WebTracerProvider({ resource });

  // register OTLP exporter, if enabled
  if (options.otlpExporter !== '') {
    provider.addSpanProcessor(
      new BatchSpanProcessor(
        new OTLPTraceExporter({
          url: options.otlpExporter,
          headers: {}
        })
      )
    );
  }

  // register context manager and propagators
  let registrationConf = {
    propagator: new CompositePropagator({
      propagators: [
        new W3CBaggagePropagator(),
        new W3CTraceContextPropagator(),
        new SentryPropagator()
      ]
    })
  };
  if (options.useZoneContext) {
    registrationConf.contextManager = new ZoneContextManager();
  }
  provider.register(registrationConf);

  // setup Sentry integration if enabled
  if (options.sentry.enabled) {
    // @ts-ignore
    provider.addSpanProcessor(new SentrySpanProcessor());
    setupSentry(options.sentry);
  }

  // register instrumentations
  registerInstrumentations({
    tracerProvider: provider,
    instrumentations: [
      new DocumentLoadInstrumentation(),
      new XMLHttpRequestInstrumentation({
        propagateTraceHeaderCorsUrls: options.propagateTraceHeaderCorsUrls
      }),
      new FetchInstrumentation({
        clearTimingResources: true,
        propagateTraceHeaderCorsUrls: options.propagateTraceHeaderCorsUrls
      })
    ]
  });
}
