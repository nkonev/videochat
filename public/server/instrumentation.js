import { NodeSDK } from '@opentelemetry/sdk-node';
import { Resource } from '@opentelemetry/resources';
import {
    ATTR_SERVICE_NAME,
    ATTR_SERVICE_VERSION,
} from '@opentelemetry/semantic-conventions';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';

const collectorOptions = {
    // use an env var OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=https://trace-service:4318/v1/traces
    // url: 'http://localhost:4318/v1/traces', // url is optional and can be omitted - default is http://localhost:4318/v1/traces
    headers: { }, // an optional object containing custom headers to be sent with each request will only work with http
    concurrencyLimit: 10, // an optional limit on pending requests
};

// https://www.npmjs.com/package/@opentelemetry/exporter-trace-otlp-http
const exporter = new OTLPTraceExporter(collectorOptions);

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
});

sdk.start();

process.on("SIGTERM", () => {
    sdk
        .shutdown()
        .then(() => console.log("Tracing terminated"))
})
