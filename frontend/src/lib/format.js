/** Formatting helpers shared by every view. */

const BINARY_UNITS = ['B', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB'];
const DECIMAL_UNITS = ['B', 'kB', 'MB', 'GB', 'TB', 'PB'];

/**
 * Render a byte count using binary or decimal units.
 *
 * @param {number} bytes
 * @param {{ base?: 'binary'|'decimal', precision?: number, compact?: boolean }} [opts]
 */
export function formatSize(bytes, opts = {}) {
  const base = opts.base === 'decimal' ? 1000 : 1024;
  const units = base === 1000 ? DECIMAL_UNITS : BINARY_UNITS;
  const precision = opts.precision ?? 1;
  if (bytes === null || bytes === undefined || Number.isNaN(bytes)) return '--';
  const value = Number(bytes);
  if (value < base) return `${Math.round(value)} ${units[0]}`;
  let index = 0;
  let scaled = value;
  while (scaled >= base && index < units.length - 1) {
    scaled /= base;
    index += 1;
  }
  const digits = scaled >= 100 ? 0 : scaled >= 10 ? Math.min(1, precision) : precision;
  const text = scaled.toFixed(digits);
  // "1.0 MiB" reads better than "1 MiB" only when there is a fraction.
  return `${digits === 0 ? text : text.replace(/\.0$/, '')} ${units[index]}`;
}

/** Compact size for dense columns: "1.2M" instead of "1.2 MiB". */
export function formatSizeShort(bytes, base = 1024) {
  if (bytes === null || bytes === undefined) return '--';
  const step = base === 1000 ? 1000 : 1024;
  const suffixes = ['', 'K', 'M', 'G', 'T', 'P'];
  const value = Number(bytes);
  if (value < step) return String(Math.round(value));
  let index = 0;
  let scaled = value;
  while (scaled >= step && index < suffixes.length - 1) {
    scaled /= step;
    index += 1;
  }
  return `${scaled < 10 ? scaled.toFixed(1).replace(/\.0$/, '') : Math.round(scaled)}${suffixes[index]}`;
}

const pad = (n) => String(n).padStart(2, '0');

/**
 * Render a timestamp using one of the supported presets.
 *
 * Presets: auto, relative, iso, time, long, short, full.
 */
export function formatDate(value, preset = 'auto') {
  if (!value) return '--';
  const date = value instanceof Date ? value : new Date(Number(value));
  if (Number.isNaN(date.getTime())) return '--';

  const now = Date.now();
  const diff = now - date.getTime();
  const minute = 60000;
  const hour = 60 * minute;
  const day = 24 * hour;

  switch (preset) {
    case 'relative': {
      if (diff < minute) return 'just now';
      if (diff < hour) return `${Math.floor(diff / minute)} min ago`;
      if (diff < day) return `${Math.floor(diff / hour)} h ago`;
      if (diff < 7 * day) return `${Math.floor(diff / day)} d ago`;
      break;
    }
    case 'iso':
      return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
    case 'time':
      return `${pad(date.getHours())}:${pad(date.getMinutes())}`;
    case 'long':
      return date.toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' });
    case 'full':
      return date.toLocaleString(undefined, {
        year: 'numeric', month: 'short', day: 'numeric',
        hour: '2-digit', minute: '2-digit', second: '2-digit',
      });
    default:
      break;
  }

  // Auto: show a time for today, a weekday for this week, otherwise a date.
  if (diff < day && date.getDate() === new Date().getDate()) {
    return `Today ${pad(date.getHours())}:${pad(date.getMinutes())}`;
  }
  if (diff < day) return `Yesterday ${pad(date.getHours())}:${pad(date.getMinutes())}`;
  if (diff < 7 * day) {
    return date.toLocaleDateString(undefined, { weekday: 'short' }) +
      ` ${pad(date.getHours())}:${pad(date.getMinutes())}`;
  }
  if (date.getFullYear() === new Date().getFullYear()) {
    return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
  }
  return date.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
}

/** Human readable duration, used for transfer ETAs. */
export function formatDuration(seconds) {
  if (seconds === null || seconds === undefined || seconds < 0) return '--';
  const s = Math.round(seconds);
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ${pad(s % 60)}s`;
  const h = Math.floor(m / 60);
  return `${h}h ${pad(m % 60)}m`;
}

/**
 * Split a filename into its stem and extension so the list can dim the
 * extension, the way modern file managers do.
 */
export function splitName(name) {
  const index = name.lastIndexOf('.');
  if (index <= 0 || index === name.length - 1) return { stem: name, ext: '' };
  return { stem: name.slice(0, index), ext: name.slice(index) };
}

/** Describe a kind for the type column. */
export function describeKind(entry) {
  if (entry.isDir) return 'Folder';
  if (entry.isSymlink && entry.linkBroken) return 'Broken link';
  if (entry.isSymlink) return 'Link to ' + (entry.target ? entry.target.split('/').pop() : 'target');
  const ext = (entry.ext || '').replace(/^\./, '').toUpperCase();
  const kindNames = {
    image: 'Image', video: 'Video', audio: 'Audio', archive: 'Archive',
    document: 'Document', code: 'Source', text: 'Text', font: 'Font',
    executable: 'Executable',
  };
  const base = kindNames[entry.kind] || 'File';
  return ext ? `${base} (${ext})` : base;
}

/** Pluralise a count: pluralise(1, 'item') -> '1 item'. */
export function pluralise(count, singular, plural) {
  const word = count === 1 ? singular : plural || `${singular}s`;
  return `${count.toLocaleString()} ${word}`;
}

/** Shorten a long path for the status bar and tooltips. */
export function shortenPath(path, max = 60) {
  if (!path || path.length <= max) return path || '';
  const parts = path.split('/').filter(Boolean);
  if (parts.length <= 3) return path;
  return `/${parts[0]}/…/${parts.slice(-2).join('/')}`;
}

/** Escape a string for use inside a regular expression. */
export function escapeRegExp(text) {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/** Clamp a number into a range. */
export function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}
