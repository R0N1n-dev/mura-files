/**
 * Thin facade over the generated Wails bindings.
 *
 * Keeping every backend call behind this module means components never need to
 * know the generated module layout, and it gives us one place to normalise
 * errors, default arguments and the thumbnail/preview URLs.
 */

import * as AppService from '../../bindings/github.com/R0N1n-dev/mura-files/internal/services/appservice.js';
import * as FileService from '../../bindings/github.com/R0N1n-dev/mura-files/internal/services/fileservice.js';
import * as OperationService from '../../bindings/github.com/R0N1n-dev/mura-files/internal/services/operationservice.js';
import * as SearchService from '../../bindings/github.com/R0N1n-dev/mura-files/internal/services/searchservice.js';
import * as WatchService from '../../bindings/github.com/R0N1n-dev/mura-files/internal/services/watchservice.js';
import { ListOptions, SearchQuery, TransferOptions, DeleteOptions } from '../../bindings/github.com/R0N1n-dev/mura-files/internal/model/index.js';

export { AppService, FileService, OperationService, SearchService, WatchService };

/* ------------------------------------------------------------------------ */
/* Defaults                                                                  */
/* ------------------------------------------------------------------------ */

export const DEFAULT_LIST_OPTIONS = {
  showHidden: false,
  filter: '',
  sortBy: 'name',
  sortDesc: false,
  dirsFirst: true,
  groupByKind: false,
  maxEntries: 0,
  recursive: false,
  depth: 0,
};

export const DEFAULT_TRANSFER_OPTIONS = {
  conflict: 'rename',
  overwrite: false,
  preserveAttrs: true,
  followSymlinks: false,
  removeSource: false,
  skipErrors: true,
};

export const DEFAULT_DELETE_OPTIONS = {
  permanent: false,
  skipErrors: true,
};

/** Build a fully populated SearchQuery so Go never sees undefined fields. */
export function searchQuery(overrides = {}) {
  return Object.assign(new SearchQuery({
    roots: [],
    pattern: '',
    mode: 'glob',
    matchCase: false,
    includeFiles: true,
    includeDirs: true,
    includeHidden: false,
    followLinks: false,
    kindFilter: '',
    extFilter: [],
    minSize: 0,
    maxSize: 0,
    modifiedAfter: 0,
    modifiedBefore: 0,
    content: false,
    maxResults: 5000,
    maxDepth: 0,
    timeoutSeconds: 0,
  }), overrides);
}

/** Build list options from a tab's view state. */
export function listOptions(state) {
  return new ListOptions({
    showHidden: !!state.showHidden,
    filter: state.filter || '',
    sortBy: state.sortBy || 'name',
    sortDesc: !!state.sortDesc,
    dirsFirst: state.dirsFirst !== false,
    groupByKind: !!state.groupByKind,
    maxEntries: state.maxEntries || 0,
    recursive: !!state.recursive,
    depth: state.depth || 0,
  });
}

export function transferOptions(overrides = {}) {
  return new TransferOptions(Object.assign({}, DEFAULT_TRANSFER_OPTIONS, overrides));
}

export function deleteOptions(overrides = {}) {
  return new DeleteOptions(Object.assign({}, DEFAULT_DELETE_OPTIONS, overrides));
}

/* ------------------------------------------------------------------------ */
/* Private asset routes served by the Go side                                */
/* ------------------------------------------------------------------------ */

/** URL that streams a file to the webview (used by the preview pane). */
export function fileUrl(path, download = false) {
  const route = download ? 'download' : 'file';
  return `/__mura/${route}?path=${encodeURIComponent(path)}`;
}

/** URL for a cached thumbnail. */
export function thumbUrl(path, size = 128) {
  return `/__mura/thumb?path=${encodeURIComponent(path)}&size=${size}`;
}

/** URL for a themed desktop icon. */
export function iconUrl(name) {
  return `/__mura/icon?name=${encodeURIComponent(name)}`;
}

/* ------------------------------------------------------------------------ */
/* Error normalisation                                                       */
/* ------------------------------------------------------------------------ */

/** Turn whatever the binding layer throws into a readable string. */
export function describeError(error) {
  if (!error) return 'Unknown error';
  if (typeof error === 'string') return error;
  if (error.message) return error.message;
  try {
    return JSON.stringify(error);
  } catch {
    return String(error);
  }
}

/**
 * Wrap a binding call so callers always get a resolved value plus an error
 * string, which keeps the components free of try/catch noise.
 */
export async function attempt(promise) {
  try {
    const value = await promise;
    return { value, error: '' };
  } catch (error) {
    return { value: null, error: describeError(error) };
  }
}
