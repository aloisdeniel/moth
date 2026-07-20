import type { MessageInitShape } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import type { TimestampSchema } from "@bufbuild/protobuf/wkt";

// Demo state is plain JSON (localStorage), so instants are stored as epoch
// milliseconds and converted to protobuf Timestamps at the RPC boundary.

export type Millis = number;

export function ts(millis: Millis): MessageInitShape<typeof TimestampSchema> {
  return {
    seconds: BigInt(Math.floor(millis / 1000)),
    nanos: Math.floor(millis % 1000) * 1e6,
  };
}

export function now(): Millis {
  return Date.now();
}

export function daysAgo(days: number): Millis {
  return Date.now() - days * 24 * 60 * 60 * 1000;
}

export function minutesAgo(minutes: number): Millis {
  return Date.now() - minutes * 60 * 1000;
}

// randomId mints ids for objects created interactively in the demo (seeded
// objects use the deterministic ids from ids.ts).
export function randomId(): string {
  return crypto.randomUUID();
}

export function notFound(what: string): ConnectError {
  return new ConnectError(`${what} not found`, Code.NotFound);
}

export function invalidArgument(message: string): ConnectError {
  return new ConnectError(message, Code.InvalidArgument);
}

export function unauthenticated(): ConnectError {
  return new ConnectError("not signed in", Code.Unauthenticated);
}

export function failedPrecondition(message: string): ConnectError {
  return new ConnectError(message, Code.FailedPrecondition);
}
