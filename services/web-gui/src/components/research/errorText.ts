import { ApiError, GENERIC_CONNECTION_ERROR } from "../../api/client";

/** The server's own words for why a request failed, or the generic connection message. */
export function errorText(err: unknown): string {
  return err instanceof ApiError && err.message ? err.message : GENERIC_CONNECTION_ERROR;
}
