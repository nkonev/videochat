import { NodeSDK } from '@opentelemetry/sdk-node';
import { Resource } from '@opentelemetry/resources';
import {
    ATTR_SERVICE_NAME,
    ATTR_SERVICE_VERSION,
} from '@opentelemetry/semantic-conventions';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { HttpInstrumentation } from '@opentelemetry/instrumentation-http';
import { ExpressInstrumentation } from '@opentelemetry/instrumentation-express';
import { AlwaysOnSampler, BatchSpanProcessor } from '@opentelemetry/sdk-trace-base';
import { JaegerPropagator } from '@opentelemetry/propagator-jaeger';
import { WinstonInstrumentation } from "@opentelemetry/instrumentation-winston";

const collectorOptions = {
    // use an env var OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=https://trace-service:4318/v1/traces
    // url: 'http://localhost:4318/v1/traces', // url is optional and can be omitted - default is http://localhost:4318/v1/traces
    headers: { }, // an optional object containing custom headers to be sent with each request will only work with http
    concurrencyLimit: 10, // an optional limit on pending requests
};

// https://www.npmjs.com/package/@opentelemetry/exporter-trace-otlp-http
const exporter = new OTLPTraceExporter(collectorOptions);

const processor = new BatchSpanProcessor(exporter);

// https://github.com/open-telemetry/opentelemetry-js-contrib/blob/main/examples/express/src/tracer.ts
// https://github.com/open-telemetry/opentelemetry-js/issues/2936
// https://www.freecodecamp.org/news/how-to-use-opentelementry-to-trace-node-js-applications/
// https://opentelemetry.io/docs/languages/js/instrumentation/
const sdk = new NodeSDK({
    resource: new Resource({
        [ATTR_SERVICE_NAME]: 'public',
        [ATTR_SERVICE_VERSION]: '0.0.0',
    }),
    // metricReader: new PeriodicExportingMetricReader({
    //     exporter: new ConsoleMetricExporter(),
    // }),
    spanProcessor: processor,
    sampler: new AlwaysOnSampler(),
    instrumentations: [
        // Express instrumentation expects HTTP layer to be instrumented
        new HttpInstrumentation({
            ignoreIncomingRequestHook(req) {
                // Ignore spans from static assets.
                return ignoreTrace(req);
            }
        }),
        new ExpressInstrumentation(),
        // https://github.com/open-telemetry/opentelemetry-js-contrib/tree/main/plugins/node/opentelemetry-instrumentation-winston
        new WinstonInstrumentation({
            // See below for Winston instrumentation options.
            disableLogSending: true
        }),
    ],
    // https://www.npmjs.com/package/@opentelemetry/propagator-jaeger
    textMapPropagator: new JaegerPropagator(),
});

sdk.start();

process.on("SIGTERM", () => {
    sdk
        .shutdown()
        .then(() => console.log("Tracing terminated"))
})

function ignoreTrace(req) {
    return req.url === "/health" ||
        req.url?.startsWith('/public/assets') ||
        req.url?.startsWith('/public/node_modules')
        ;
}
