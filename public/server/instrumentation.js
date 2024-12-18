import { NodeSDK } from '@opentelemetry/sdk-node';
// const { ConsoleSpanExporter } = require('@opentelemetry/sdk-trace-node');
// const {
//     PeriodicExportingMetricReader,
//     ConsoleMetricExporter,
// } = require('@opentelemetry/sdk-metrics');
import { Resource } from '@opentelemetry/resources';
import {
    ATTR_SERVICE_NAME,
    ATTR_SERVICE_VERSION,
} from '@opentelemetry/semantic-conventions';
// const { BasicTracerProvider, BatchSpanProcessor } = require('@opentelemetry/sdk-trace-base');
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';

const collectorOptions = {
    url: 'http://localhost:4318/v1/traces', // url is optional and can be omitted - default is http://localhost:4318/v1/traces
    headers: { }, // an optional object containing custom headers to be sent with each request will only work with http
    concurrencyLimit: 10, // an optional limit on pending requests
};

const exporter = new OTLPTraceExporter(collectorOptions);
// const provider = new BasicTracerProvider({
//     spanProcessors: [
//         new BatchSpanProcessor(exporter, {
//             // The maximum queue size. After the size is reached spans are dropped.
//             maxQueueSize: 1000,
//             // The interval between two consecutive exports
//             scheduledDelayMillis: 30000,
//         })
//     ]
// });
// provider.register();

const sdk = new NodeSDK({
    resource: new Resource({
        [ATTR_SERVICE_NAME]: 'public',
        [ATTR_SERVICE_VERSION]: '0.0.0',
    }),
    // traceExporter: new ConsoleSpanExporter(),
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
