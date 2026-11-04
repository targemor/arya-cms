/**
 * Base path of the Go API. Requests stay same-origin: in development `ng serve` proxies it to the
 * backend (see proxy.conf.js), in production the reverse proxy in front of both does.
 */
export const API_BASE = '/v1';

/** Error body returned by every API endpoint. */
export interface ApiError {
  error: string;
}

/** Strapi-style pagination metadata returned with every list. */
export interface Pagination {
  page: number;
  pageSize: number;
  pageCount: number;
  total: number;
}

/** A page of a list endpoint: `{ data, meta: { pagination } }`. */
export interface Page<T> {
  data: T[];
  meta: { pagination: Pagination };
}
