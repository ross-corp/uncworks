import { NodeTracerProvider } from "@opentelemetry/sdk-trace-node";
import { SimpleSpanProcessor } from "@opentelemetry/sdk-trace-node";
import { OTLPTraceExporter } from "@opentelemetry/exporter-trace-otlp-grpc";
import { resourceFromAttributes } from "@opentelemetry/resources";
import type { SpanExporter } from "@opentelemetry/sdk-trace-node";

export interface TracingConfig {
  serviceName: string;
  agentRunId: string;
  collectorEndpoint?: string;
  exporter?: SpanExporter;
}

/** Initialize OpenTelemetry tracing for the agent harness. */
export function initTracing(config: TracingConfig): NodeTracerProvider {
  const resource = resourceFromAttributes({
    "service.name": config.serviceName,
    "agent_run.id": config.agentRunId,
  });

  const exporter =
    config.exporter ??
    new OTLPTraceExporter({
      url: config.collectorEndpoint ?? "http://localhost:4317",
    });

  // SDK 2.x takes span processors in the constructor. addSpanProcessor is gone.
  const provider = new NodeTracerProvider({
    resource,
    spanProcessors: [new SimpleSpanProcessor(exporter)],
  });
  provider.register();

  return provider;
}

/** Shut down the tracing provider. */
export async function shutdownTracing(
  provider: NodeTracerProvider
): Promise<void> {
  await provider.shutdown();
}
