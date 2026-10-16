/**
 * Icon set used across the interface.
 *
 * Every entry is the inner markup of a 24x24 stroke icon. The <Icon> component
 * wraps it with the shared stroke attributes so the set stays consistent and
 * easy to extend.
 *
 * Shapes follow the Lucide icon language (24px grid, 1.7px stroke, round caps)
 * which is what gives the interface its calm, modern feel.
 */

export const ICONS = {
  /* ----------------------------- Navigation ----------------------------- */
  'chevron-left': '<path d="M15 18l-6-6 6-6"/>',
  'chevron-right': '<path d="M9 18l6-6-6-6"/>',
  'chevron-down': '<path d="M6 9l6 6 6-6"/>',
  'chevron-up': '<path d="M18 15l-6-6-6 6"/>',
  'arrow-left': '<path d="M19 12H5"/><path d="M12 19l-7-7 7-7"/>',
  'arrow-right': '<path d="M5 12h14"/><path d="M12 5l7 7-7 7"/>',
  'arrow-up': '<path d="M12 19V5"/><path d="M5 12l7-7 7 7"/>',
  'arrow-down': '<path d="M12 5v14"/><path d="M19 12l-7 7-7-7"/>',
  'corner-up-left': '<path d="M9 14L4 9l5-5"/><path d="M20 20v-7a4 4 0 0 0-4-4H4"/>',
  'home': '<path d="M3 10.5L12 3l9 7.5"/><path d="M5 9.5V21h14V9.5"/><path d="M9.5 21v-6h5v6"/>',
  'external-link': '<path d="M14 3h7v7"/><path d="M21 3l-9 9"/><path d="M19 14v5a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2h5"/>',

  /* ------------------------------- Actions ------------------------------- */
  'search': '<circle cx="11" cy="11" r="7"/><path d="M20 20l-3.6-3.6"/>',
  'plus': '<path d="M12 5v14"/><path d="M5 12h14"/>',
  'minus': '<path d="M5 12h14"/>',
  'x': '<path d="M18 6L6 18"/><path d="M6 6l12 12"/>',
  'check': '<path d="M20 6L9 17l-5-5"/>',
  'refresh': '<path d="M21 12a9 9 0 1 1-2.64-6.36"/><path d="M21 4v5h-5"/>',
  'folder-plus': '<path d="M4 20a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h4l2 2.5h8a2 2 0 0 1 2 2V18a2 2 0 0 1-2 2z"/><path d="M12 11v5"/><path d="M9.5 13.5h5"/>',
  'file-plus': '<path d="M14 2H7a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V7z"/><path d="M14 2v5h5"/><path d="M12 12v5"/><path d="M9.5 14.5h5"/>',
  'trash': '<path d="M4 7h16"/><path d="M10 11v6"/><path d="M14 11v6"/><path d="M6 7l1 12a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-12"/><path d="M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/>',
  'trash-2': '<path d="M4 7h16"/><path d="M10 11v6"/><path d="M14 11v6"/><path d="M6 7l1 12a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-12"/><path d="M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/><path d="M3 3l18 18"/>',
  'restore': '<path d="M3 12a9 9 0 1 0 9-9 9 9 0 0 0-6.36 2.64L3 8"/><path d="M3 3v5h5"/>',
  'copy': '<rect x="9" y="9" width="12" height="12" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>',
  'scissors': '<circle cx="6" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="M20 4L8.12 15.88"/><path d="M14.47 14.48L20 20"/><path d="M8.12 8.12L12 12"/>',
  'clipboard': '<rect x="8" y="2" width="8" height="4" rx="1"/><path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/>',
  'paste': '<path d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2"/><rect x="8" y="2" width="8" height="4" rx="1"/><path d="M9 14h6"/><path d="M9 17h4"/>',
  'pencil': '<path d="M4 20h4l10-10a2.83 2.83 0 0 0-4-4L4 16z"/><path d="M14.5 5.5l4 4"/>',
  'download': '<path d="M12 3v12"/><path d="M7 11l5 5 5-5"/><path d="M4 20h16"/>',
  'upload': '<path d="M12 21V9"/><path d="M7 13l5-5 5 5"/><path d="M4 4h16"/>',
  'save': '<path d="M4 4h11l5 5v11a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1z"/><path d="M8 4v5h7"/><path d="M8 20v-6h8v6"/>',
  'share': '<circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><path d="M8.6 13.5l6.8 4"/><path d="M15.4 6.5l-6.8 4"/>',
  'link': '<path d="M10 13a5 5 0 0 0 7.5.5l3-3a5 5 0 0 0-7-7l-1.7 1.7"/><path d="M14 11a5 5 0 0 0-7.5-.5l-3 3a5 5 0 0 0 7 7l1.7-1.7"/>',
  'eye': '<path d="M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7-10-7-10-7z"/><circle cx="12" cy="12" r="3"/>',
  'eye-off': '<path d="M10.6 6.2A9.9 9.9 0 0 1 12 6c6.4 0 10 6 10 6a17 17 0 0 1-3.2 4"/><path d="M6.2 6.7A17 17 0 0 0 2 12s3.6 6 10 6a9.6 9.6 0 0 0 4.4-1"/><path d="M3 3l18 18"/><path d="M9.9 9.9a3 3 0 0 0 4.2 4.2"/>',
  'settings': '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-2.9 1.2 2 2 0 1 1-4 0 1.7 1.7 0 0 0-2.9-1.2l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1A1.7 1.7 0 0 0 2.6 15a2 2 0 1 1 0-4 1.7 1.7 0 0 0 1.2-2.9l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1A1.7 1.7 0 0 0 9.5 4a2 2 0 1 1 4 0 1.7 1.7 0 0 0 2.9 1.2l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1A1.7 1.7 0 0 0 21.4 11a2 2 0 1 1 0 4z"/>',
  'sliders': '<path d="M4 6h10"/><path d="M18 6h2"/><path d="M4 12h4"/><path d="M12 12h8"/><path d="M4 18h12"/><path d="M20 18h0"/><circle cx="16" cy="6" r="2"/><circle cx="10" cy="12" r="2"/><circle cx="18" cy="18" r="2"/>',
  'more-horizontal': '<circle cx="5" cy="12" r="1.6"/><circle cx="12" cy="12" r="1.6"/><circle cx="19" cy="12" r="1.6"/>',
  'more-vertical': '<circle cx="12" cy="5" r="1.6"/><circle cx="12" cy="12" r="1.6"/><circle cx="12" cy="19" r="1.6"/>',
  'info': '<circle cx="12" cy="12" r="9"/><path d="M12 16v-5"/><path d="M12 8h.01"/>',
  'alert-triangle': '<path d="M10.3 3.9L1.9 18a2 2 0 0 0 1.7 3h16.8a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z"/><path d="M12 9v4"/><path d="M12 17h.01"/>',
  'alert-circle': '<circle cx="12" cy="12" r="9"/><path d="M12 8v4"/><path d="M12 16h.01"/>',
  'check-circle': '<circle cx="12" cy="12" r="9"/><path d="M8.5 12.5l2.5 2.5 4.5-5"/>',
  'x-circle': '<circle cx="12" cy="12" r="9"/><path d="M15 9l-6 6"/><path d="M9 9l6 6"/>',
  'help': '<circle cx="12" cy="12" r="9"/><path d="M9.6 9.3A2.5 2.5 0 1 1 12 13v1"/><path d="M12 17h.01"/>',
  'zap': '<path d="M13 2L4 14h7l-1 8 9-12h-7z"/>',
  'sparkles': '<path d="M12 3l1.6 4.4L18 9l-4.4 1.6L12 15l-1.6-4.4L6 9l4.4-1.6z"/><path d="M18.5 15l.8 2.2 2.2.8-2.2.8-.8 2.2-.8-2.2-2.2-.8 2.2-.8z"/>',
  'loader': '<path d="M12 3v4"/><path d="M12 17v4"/><path d="M5.6 5.6l2.8 2.8"/><path d="M15.6 15.6l2.8 2.8"/><path d="M3 12h4"/><path d="M17 12h4"/><path d="M5.6 18.4l2.8-2.8"/><path d="M15.6 8.4l2.8-2.8"/>',
  'filter': '<path d="M3 5h18l-7 8v6l-4 2v-8z"/>',
  'history': '<path d="M3 12a9 9 0 1 0 3-6.7L3 8"/><path d="M3 3v5h5"/><path d="M12 8v4l3 2"/>',
  'clock': '<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>',
  'calendar': '<rect x="3" y="5" width="18" height="16" rx="2"/><path d="M3 10h18"/><path d="M8 3v4"/><path d="M16 3v4"/>',
  'hash': '<path d="M5 9h14"/><path d="M5 15h14"/><path d="M10 3l-1.5 18"/><path d="M15.5 3L14 21"/>',
  'tag': '<path d="M20.6 13.4l-7.2 7.2a2 2 0 0 1-2.8 0l-7.2-7.2a2 2 0 0 1-.6-1.4V4a1 1 0 0 1 1-1h8a2 2 0 0 1 1.4.6l7.4 7.4a2 2 0 0 1 0 2.8z"/><path d="M7.5 7.5h.01"/>',
  'star': '<path d="M12 3.5l2.6 5.3 5.9.9-4.3 4.1 1 5.8-5.2-2.8-5.2 2.8 1-5.8-4.3-4.1 5.9-.9z"/>',
  'bookmark': '<path d="M6 3h12a1 1 0 0 1 1 1v17l-7-4-7 4V4a1 1 0 0 1 1-1z"/>',
  'lock': '<rect x="4" y="10" width="16" height="11" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/>',
  'unlock': '<rect x="4" y="10" width="16" height="11" rx="2"/><path d="M8 10V7a4 4 0 0 1 7.5-2"/>',
  'shield': '<path d="M12 3l8 3v6c0 5-3.4 8.4-8 10-4.6-1.6-8-5-8-10V6z"/>',
  'terminal': '<rect x="2" y="4" width="20" height="16" rx="2"/><path d="M6 9l3 3-3 3"/><path d="M13 15h5"/>',
  'keyboard': '<rect x="2" y="6" width="20" height="12" rx="2"/><path d="M6 10h.01"/><path d="M10 10h.01"/><path d="M14 10h.01"/><path d="M18 10h.01"/><path d="M8 14h8"/>',
  'palette': '<path d="M12 3a9 9 0 1 0 0 18 2 2 0 0 0 1.6-3.2 2 2 0 0 1 1.6-3.2H18a3 3 0 0 0 3-3A9 9 0 0 0 12 3z"/><circle cx="7.5" cy="10.5" r="1.2"/><circle cx="12" cy="7.5" r="1.2"/><circle cx="16.5" cy="10.5" r="1.2"/><circle cx="7.5" cy="15" r="1.2"/>',
  'sun': '<circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="M4.9 4.9l1.4 1.4"/><path d="M17.7 17.7l1.4 1.4"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="M4.9 19.1l1.4-1.4"/><path d="M17.7 6.3l1.4-1.4"/>',
  'moon': '<path d="M20 14.5A8.5 8.5 0 0 1 9.5 4a8.5 8.5 0 1 0 10.5 10.5z"/>',
  'monitor': '<rect x="2" y="4" width="20" height="13" rx="2"/><path d="M8 21h8"/><path d="M12 17v4"/>',
  'layout-list': '<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M8 9h9"/><path d="M8 13h9"/><path d="M8 17h5"/>',
  'layout-grid': '<rect x="3" y="3" width="7.5" height="7.5" rx="1.5"/><rect x="13.5" y="3" width="7.5" height="7.5" rx="1.5"/><rect x="3" y="13.5" width="7.5" height="7.5" rx="1.5"/><rect x="13.5" y="13.5" width="7.5" height="7.5" rx="1.5"/>',
  'layout-columns': '<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M9 4v16"/><path d="M15 4v16"/>',
  'panel-left': '<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M9 4v16"/>',
  'panel-right': '<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M15 4v16"/>',
  'sort-asc': '<path d="M7 4v16"/><path d="M4 7l3-3 3 3"/><path d="M13 6h7"/><path d="M13 12h5"/><path d="M13 18h3"/>',
  'sort-desc': '<path d="M7 20V4"/><path d="M4 17l3 3 3-3"/><path d="M13 6h7"/><path d="M13 12h5"/><path d="M13 18h3"/>',
  'play': '<path d="M7 4.5l12 7.5-12 7.5z"/>',
  'pause': '<rect x="7" y="5" width="3.5" height="14" rx="1"/><rect x="13.5" y="5" width="3.5" height="14" rx="1"/>',
  'maximise': '<rect x="4" y="4" width="16" height="16" rx="2"/>',
  'restore-window': '<rect x="7" y="3" width="14" height="14" rx="2"/><path d="M17 17v2a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V9a2 2 0 0 1 2-2h2"/>',
  'minimise': '<path d="M5 12h14"/>',
};

/* ------------------------------------------------------------------------ */
/* Filesystem icons                                                          */
/* ------------------------------------------------------------------------ */

const FOLDER = '<path d="M4 20a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h4l2 2.5h8a2 2 0 0 1 2 2V18a2 2 0 0 1-2 2z"/>';
const PAGE = '<path d="M14 2H7a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V7z"/><path d="M14 2v5h5"/>';

export const FILE_ICONS = {
  folder: FOLDER,
  'folder-open': FOLDER + '<path d="M2 14h20"/>',
  'folder-symlink': FOLDER + '<path d="M10 13h4"/><path d="M12 11l2 2-2 2"/>',
  file: PAGE,
  'file-text': PAGE + '<path d="M9 13h6"/><path d="M9 17h6"/>',
  'file-document': PAGE + '<path d="M9 13h6"/><path d="M9 17h4"/>',
  'file-pdf': PAGE + '<path d="M9 13h6"/><path d="M9 17h6"/><path d="M9 9h1.5"/>',
  'file-word': PAGE + '<path d="M9 13l1.5 5 1.5-4 1.5 4 1.5-5"/>',
  'file-excel': PAGE + '<path d="M9.5 13l5 5"/><path d="M14.5 13l-5 5"/>',
  'file-slides': PAGE + '<rect x="9" y="13" width="6" height="5" rx="1"/>',
  'file-code': PAGE + '<path d="M10 13l-1.5 2.5L10 18"/><path d="M14 13l1.5 2.5L14 18"/>',
  'file-shell': PAGE + '<path d="M9.5 13.5l2 2-2 2"/><path d="M13 17.5h2.5"/>',
  'file-json': PAGE + '<path d="M10 13.5c-1 0-1 .8-1 1.5s0 1.5-1 1.5"/><path d="M14 13.5c1 0 1 .8 1 1.5s0 1.5 1 1.5"/>',
  'file-markdown': PAGE + '<path d="M8.5 18v-5l2 2.2 2-2.2v5"/><path d="M15.5 13v5"/><path d="M14 16.2l1.5 1.8 1.5-1.8"/>',
  'file-image': PAGE + '<circle cx="10.5" cy="13.5" r="1.2"/><path d="M9 19l3-3.2 2.2 2.2 1.6-1.6L18 19"/>',
  'file-video': PAGE + '<path d="M10 13.5l4 2-4 2z"/>',
  'file-audio': PAGE + '<path d="M12.5 18v-5l3-1"/><circle cx="10.8" cy="18" r="1.8"/>',
  'file-archive': PAGE + '<path d="M12 11v1"/><path d="M12 14v1"/><path d="M12 17v1"/>',
  'file-package': PAGE + '<path d="M12 11l3 1.6-3 1.6-3-1.6z"/>',
  'file-disc': '<circle cx="12" cy="12" r="9"/><circle cx="12" cy="12" r="3"/>',
  'file-font': PAGE + '<path d="M10 18l2-6 2 6"/><path d="M10.6 16h2.8"/>',
  'file-desktop': PAGE + '<rect x="9" y="13" width="6" height="5" rx="1"/><path d="M12 11v2"/>',
  'file-lock': PAGE + '<rect x="9.5" y="14" width="5" height="4" rx="1"/><path d="M10.8 14v-1.2a1.2 1.2 0 0 1 2.4 0V14"/>',
  'file-exec': PAGE + '<path d="M10 14l3 2-3 2z"/>',
  'file-database': '<ellipse cx="12" cy="6" rx="8" ry="3"/><path d="M4 6v12c0 1.7 3.6 3 8 3s8-1.3 8-3V6"/><path d="M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3"/>',
};

/* ------------------------------------------------------------------------ */
/* Places, drives and miscellaneous                                          */
/* ------------------------------------------------------------------------ */

export const PLACE_ICONS = {
  home: '<path d="M3 10.5L12 3l9 7.5"/><path d="M5 9.5V21h14V9.5"/><path d="M9.5 21v-6h5v6"/>',
  desktop: '<rect x="2" y="4" width="20" height="13" rx="2"/><path d="M8 21h8"/><path d="M12 17v4"/>',
  documents: '<path d="M14 2H7a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V7z"/><path d="M14 2v5h5"/><path d="M9 13h6"/><path d="M9 17h6"/>',
  download: '<path d="M12 3v12"/><path d="M7 11l5 5 5-5"/><path d="M4 20h16"/>',
  music: '<circle cx="7" cy="17" r="2.6"/><circle cx="18" cy="15" r="2.6"/><path d="M9.6 17V7l11-2v10"/>',
  pictures: '<rect x="3" y="4" width="18" height="16" rx="2"/><circle cx="9" cy="9.5" r="1.5"/><path d="M4 17l4.5-4.5 3.5 3.5 3-3L20 17"/>',
  videos: '<rect x="2" y="5" width="14" height="14" rx="2"/><path d="M16 10l6-3.5v11L16 14z"/>',
  public: '<circle cx="12" cy="12" r="9"/><path d="M3 12h18"/><path d="M12 3c2.5 2.6 3.8 5.7 3.8 9S14.5 18.4 12 21c-2.5-2.6-3.8-5.7-3.8-9S9.5 5.6 12 3z"/>',
  templates: '<rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18"/><path d="M9 9v12"/>',
  trash: '<path d="M4 7h16"/><path d="M10 11v6"/><path d="M14 11v6"/><path d="M6 7l1 12a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-12"/><path d="M9 7V5a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v2"/>',
  root: '<path d="M3 21h18"/><path d="M5 21V8l7-5 7 5v13"/><path d="M10 21v-6h4v6"/>',
  'hard-drive': '<rect x="2" y="13" width="20" height="7" rx="2"/><path d="M4.5 13L7 5h10l2.5 8"/><path d="M7 16.5h.01"/><path d="M11 16.5h.01"/>',
  'drive-removable': '<rect x="2" y="13" width="20" height="7" rx="2"/><path d="M4.5 13L7 5h10l2.5 8"/><path d="M17 16.5h.01"/>',
  usb: '<circle cx="12" cy="20" r="2"/><path d="M12 18V8"/><path d="M9 8h6"/><path d="M12 5l-2 3h4z"/><path d="M15 11h3v2"/>',
  server: '<rect x="3" y="4" width="18" height="7" rx="2"/><rect x="3" y="13" width="18" height="7" rx="2"/><path d="M7 7.5h.01"/><path d="M7 16.5h.01"/>',
  cloud: '<path d="M17.5 19a4.5 4.5 0 0 0 .4-9A6 6 0 0 0 6.2 11.2A3.9 3.9 0 0 0 7 19z"/>',
  network: '<rect x="9" y="2" width="6" height="6" rx="1.5"/><rect x="2" y="16" width="6" height="6" rx="1.5"/><rect x="16" y="16" width="6" height="6" rx="1.5"/><path d="M12 8v4"/><path d="M5 16v-2h14v2"/>',
  bookmark: '<path d="M6 3h12a1 1 0 0 1 1 1v17l-7-4-7 4V4a1 1 0 0 1 1-1z"/>',
  folder: FOLDER,
  'folder-tree': '<path d="M3 5a2 2 0 0 1 2-2h3l2 2.5h6a2 2 0 0 1 2 2V12"/><path d="M3 5v14a2 2 0 0 0 2 2h4"/><path d="M13 20h8"/><path d="M17 20v-4h4v4"/><path d="M9 20v-5h4"/>',
  user: '<circle cx="12" cy="8" r="4"/><path d="M4 21v-1.5A5.5 5.5 0 0 1 9.5 14h5A5.5 5.5 0 0 1 20 19.5V21"/>',
  cpu: '<rect x="6" y="6" width="12" height="12" rx="2"/><path d="M9.5 2v3"/><path d="M14.5 2v3"/><path d="M9.5 19v3"/><path d="M14.5 19v3"/><path d="M2 9.5h3"/><path d="M2 14.5h3"/><path d="M19 9.5h3"/><path d="M19 14.5h3"/>',
  activity: '<path d="M3 12h4l2.5-7 4 14 2.5-7h5"/>',
  gauge: '<path d="M4 18a9 9 0 1 1 16 0"/><path d="M12 14l4-4"/>',
  'git-branch': '<circle cx="6" cy="5" r="2.4"/><circle cx="6" cy="19" r="2.4"/><circle cx="18" cy="9" r="2.4"/><path d="M6 7.4v9.2"/><path d="M15.6 9c-2 0-4.6 1-6 3"/>',
  package: '<path d="M20.5 7.3L12 12l-8.5-4.7"/><path d="M12 12v9.5"/><path d="M3.5 7.3v9.4a1.6 1.6 0 0 0 .9 1.5l7 3.3a1.6 1.6 0 0 0 1.2 0l7-3.3a1.6 1.6 0 0 0 .9-1.5V7.3a1.6 1.6 0 0 0-.9-1.5l-7-3.3a1.6 1.6 0 0 0-1.2 0l-7 3.3a1.6 1.6 0 0 0-.9 1.5z"/>',
  wrench: '<path d="M15.5 8.5a4.5 4.5 0 0 0 5.6-5.6l-2.9 2.9-2-2 2.9-2.9a4.5 4.5 0 0 0-5.6 5.6L3 16.9a2 2 0 1 0 2.8 2.8z"/>',
};

/** Resolve an icon key to markup, falling back to a generic file icon. */
export function iconMarkup(name) {
  return ICONS[name] || FILE_ICONS[name] || PLACE_ICONS[name] || FILE_ICONS.file;
}

/** Icon keys that should render as a filled shape rather than a stroke. */
export const FILLED = new Set(['star']);
