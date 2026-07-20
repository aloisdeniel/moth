import type {
  DescMessage,
  DescMethodStreaming,
  DescMethodUnary,
  MessageInitShape,
  MessageShape,
} from "@bufbuild/protobuf";
import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import type { Transport, UnaryResponse } from "@connectrpc/connect";

import { loadState, saveState } from "./state";

// The demo backend: a connect Transport whose unary calls are answered by
// in-memory handlers over a localStorage-persisted state instead of the
// network. Screens are unaware — they use the exact same generated clients.

type AnyHandler = (state: never, req: never) => unknown;

const registry = new Map<string, AnyHandler>();

function key(method: DescMethodUnary | DescMethodStreaming): string {
  return `${method.parent.typeName}/${method.name}`;
}

// handle registers the demo implementation of one unary RPC. `S` is the
// slice of the demo state the handler needs; the transport passes the full
// state, which structurally satisfies any slice. Handlers may mutate the
// state freely — it is persisted after every call — and throw ConnectError
// (via util.ts helpers) for validation and not-found cases.
export function handle<I extends DescMessage, O extends DescMessage, S>(
  method: DescMethodUnary<I, O>,
  fn: (state: S, req: MessageShape<I>) => MessageInitShape<O>,
): void {
  registry.set(key(method), fn as AnyHandler);
}

// A short artificial latency keeps spinners and optimistic states visible,
// so the demo feels like a real deployment rather than an instant mock.
function latency(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, 60 + Math.random() * 140));
}

export function createDemoTransport(): Transport {
  return {
    async unary<I extends DescMessage, O extends DescMessage>(
      method: DescMethodUnary<I, O>,
      signal: AbortSignal | undefined,
      _timeoutMs: number | undefined,
      _header: HeadersInit | undefined,
      input: MessageInitShape<I>,
    ): Promise<UnaryResponse<I, O>> {
      await latency();
      signal?.throwIfAborted();
      const fn = registry.get(key(method));
      if (!fn) {
        throw new ConnectError(
          `demo mode does not implement ${key(method)}`,
          Code.Unimplemented,
        );
      }
      const state = loadState();
      const output = (fn as (state: unknown, req: MessageShape<I>) => MessageInitShape<O>)(
        state,
        create(method.input, input),
      );
      saveState(state);
      return {
        stream: false,
        service: method.parent,
        method,
        header: new Headers(),
        trailer: new Headers(),
        message: create(method.output, output),
      };
    },
    stream() {
      return Promise.reject(
        new ConnectError("demo mode does not support streaming RPCs", Code.Unimplemented),
      );
    },
  };
}
