import { NodeSDK } from '@opentelemetry/sdk-node';
import { Resource } from '@opentelemetry/resources';
import {
    ATTR_SERVICE_NAME,
    ATTR_SERVICE_VERSION,
    ATTR_HTTP_ROUTE,
} from '@opentelemetry/semantic-conventions';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
// const { NodeTracerProvider } = require('@opentelemetry/sdk-trace-node');
// const { SEMRESATTRS_SERVICE_NAME } = require('@opentelemetry/semantic-conventions');
// const { SimpleSpanProcessor } = require('@opentelemetry/sdk-trace-base');
// import { registerInstrumentations } from '@opentelemetry/instrumentation';
import { HttpInstrumentation } from '@opentelemetry/instrumentation-http';
import { ExpressInstrumentation } from '@opentelemetry/instrumentation-express';
import { AlwaysOnSampler } from '@opentelemetry/sdk-trace-base';
import { SamplingDecision, SpanKind } from '@opentelemetry/api';
// import { NodeTracerProvider } from '@opentelemetry/sdk-trace-node';

const collectorOptions = {
    // use an env var OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=https://trace-service:4318/v1/traces
    // url: 'http://localhost:4318/v1/traces', // url is optional and can be omitted - default is http://localhost:4318/v1/traces
    headers: { }, // an optional object containing custom headers to be sent with each request will only work with http
    concurrencyLimit: 10, // an optional limit on pending requests
};

// https://www.npmjs.com/package/@opentelemetry/exporter-trace-otlp-http
const exporter = new OTLPTraceExporter(collectorOptions);

// https://github.com/open-telemetry/opentelemetry-js-contrib/blob/main/examples/express/src/tracer.ts

// const provider = new NodeTracerProvider({
//     resource: new Resource({
//         [ATTR_SERVICE_NAME]: 'public',
//         [ATTR_SERVICE_VERSION]: '0.0.0',
//     }),
//     spanProcessors: [new SimpleSpanProcessor(exporter)],
//     sampler: filterSampler(ignoreHealthCheck, new AlwaysOnSampler()),
// });
// registerInstrumentations({
//     tracerProvider: provider,
//     instrumentations: [
//         // Express instrumentation expects HTTP layer to be instrumented
//         new HttpInstrumentation(),
//         new ExpressInstrumentation(),
//     ],
// });
//
// // Initialize the OpenTelemetry APIs to use the NodeTracerProvider bindings
// provider.register();

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
    traceExporter: exporter,
    sampler: filterSampler(ignoreHealthCheck, new AlwaysOnSampler()),
    instrumentations: [
        // Express instrumentation expects HTTP layer to be instrumented
        new HttpInstrumentation(),
        new ExpressInstrumentation(),
    ],
});

sdk.start();

process.on("SIGTERM", () => {
    sdk
        .shutdown()
        .then(() => console.log("Tracing terminated"))
})

function filterSampler(filterFn, parent) {
    return {
        shouldSample(ctx, tid, spanName, spanKind, attr, links) {
            if (!filterFn(spanName, spanKind, attr)) {
                return { decision: SamplingDecision.NOT_RECORD };
            }
            return parent.shouldSample(ctx, tid, spanName, spanKind, attr, links);
        },
        toString() {
            return `FilterSampler(${parent.toString()})`;
        }
    }
}

function ignoreHealthCheck(spanName, spanKind, attributes) {
    return spanKind !== SpanKind.SERVER || attributes[ATTR_HTTP_ROUTE] !== "/health";
}