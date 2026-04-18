// Package generator converts HTML+CSS into high-fidelity DOCX and PDF outputs.
package generator

// AISafetyCSS is the UAE AI Safety print CSS profile.
//
// Visual design:
//   - 5mm Crimson Red (#BE123C) header bar repeating on every page
//   - 5mm Dark Green (#064E3B) footer bar repeating on every page
//   - 3% opacity tiled shield-logo watermark background
//   - A4 portrait, zero page margins (bars + body padding own the space)
//   - Inter / Montserrat typography for a modern government-tech aesthetic
//   - .cover-page gets page-break-after: always
const AISafetyCSS = `
/* ─── Fonts ─────────────────────────────────────────────────────────────── */
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=Montserrat:wght@600;700;800&display=swap');

/* ─── Page Setup ─────────────────────────────────────────────────────────── */
@page {
  size: A4 portrait;
  margin: 0;
}

/* ─── Base Reset ─────────────────────────────────────────────────────────── */
*, *::before, *::after {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

html {
  font-size: 10.5pt;
  line-height: 1.65;
  -webkit-print-color-adjust: exact;
  print-color-adjust: exact;
}

body {
  font-family: 'Inter', 'Segoe UI', Arial, sans-serif;
  color: #1A1A2E;
  background: #FFFFFF;
  /* top: 5mm bar + 15mm gutter = 20mm   bottom: 15mm gutter + 5mm bar = 20mm */
  padding: 20mm 20mm 20mm 20mm;
}

/* ─── Fixed Header Bar (repeats every page) ─────────────────────────────── */
.page-header {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 5mm;
  background-color: #BE123C;
  z-index: 9999;
}

/* ─── Fixed Footer Bar (repeats every page) ─────────────────────────────── */
.page-footer {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  height: 5mm;
  background-color: #064E3B;
  z-index: 9999;
}

/* ─── Watermark (tiled shield logo at 3% opacity) ───────────────────────── */
.watermark {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 120' width='80' height='96'%3E%3Cpath d='M50 6 L94 26 L94 64 C94 90 72 110 50 117 C28 110 6 90 6 64 L6 26 Z' fill='%23064E3B'/%3E%3Cpath d='M50 20 L80 35 L80 64 C80 82 67 96 50 103 C33 96 20 82 20 64 L20 35 Z' fill='%23FFFFFF'/%3E%3Cpath d='M38 60 L46 68 L64 44' stroke='%23064E3B' stroke-width='6' fill='none' stroke-linecap='round' stroke-linejoin='round'/%3E%3C/svg%3E");
  background-size: 80px 96px;
  background-repeat: repeat;
  opacity: 0.03;
  z-index: -1;
  pointer-events: none;
}

/* ─── Cover Page ─────────────────────────────────────────────────────────── */
.cover-page {
  min-height: calc(297mm - 40mm); /* A4 minus top+bottom padding */
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: flex-start;
  padding: 10mm 0;
  page-break-after: always;
  break-after: page;
}

.cover-accent {
  width: 56px;
  height: 4px;
  background: #BE123C;
  border-radius: 2px;
  margin-bottom: 10mm;
}

.cover-title {
  font-family: 'Montserrat', 'Segoe UI', Arial, sans-serif;
  font-size: 28pt;
  font-weight: 800;
  color: #1A1A2E;
  line-height: 1.15;
  margin-bottom: 6mm;
  letter-spacing: -0.02em;
}

.cover-subtitle {
  font-family: 'Inter', Arial, sans-serif;
  font-size: 13pt;
  font-weight: 400;
  color: #475569;
  margin-bottom: 14mm;
}

.cover-meta {
  margin-top: auto;
  font-size: 9pt;
  color: #64748B;
  border-top: 1px solid #E2E8F0;
  padding-top: 4mm;
  width: 100%;
}

/* ─── Headings ───────────────────────────────────────────────────────────── */
h1 {
  font-family: 'Montserrat', 'Segoe UI', Arial, sans-serif;
  font-size: 16pt;
  font-weight: 700;
  color: #BE123C;
  margin-top: 9mm;
  margin-bottom: 3mm;
  padding-bottom: 1.5mm;
  border-bottom: 2px solid #BE123C;
  page-break-after: avoid;
  break-after: avoid;
}

h2 {
  font-family: 'Montserrat', 'Segoe UI', Arial, sans-serif;
  font-size: 13pt;
  font-weight: 700;
  color: #064E3B;
  margin-top: 7mm;
  margin-bottom: 2.5mm;
  page-break-after: avoid;
  break-after: avoid;
}

h3 {
  font-family: 'Inter', Arial, sans-serif;
  font-size: 11.5pt;
  font-weight: 600;
  color: #1A1A2E;
  margin-top: 5mm;
  margin-bottom: 2mm;
  page-break-after: avoid;
  break-after: avoid;
}

h4, h5, h6 {
  font-family: 'Inter', Arial, sans-serif;
  font-size: 10.5pt;
  font-weight: 600;
  color: #334155;
  margin-top: 3.5mm;
  margin-bottom: 1.5mm;
  page-break-after: avoid;
  break-after: avoid;
}

/* ─── Body Text ──────────────────────────────────────────────────────────── */
p {
  margin-bottom: 3mm;
  text-align: justify;
  orphans: 3;
  widows: 3;
}

/* ─── Lists ──────────────────────────────────────────────────────────────── */
ul, ol {
  padding-left: 7mm;
  margin-bottom: 3mm;
}

li {
  margin-bottom: 1.2mm;
  line-height: 1.6;
}

ul li::marker { color: #BE123C; }
ol li::marker { color: #064E3B; font-weight: 600; }

/* ─── Tables ─────────────────────────────────────────────────────────────── */
table {
  width: 100%;
  border-collapse: collapse;
  margin: 4mm 0 5mm;
  font-size: 9.5pt;
  page-break-inside: avoid;
  break-inside: avoid;
}

thead th {
  background-color: #BE123C;
  color: #FFFFFF;
  font-family: 'Montserrat', sans-serif;
  font-weight: 700;
  font-size: 8.5pt;
  padding: 2.2mm 3mm;
  text-align: left;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

tbody tr:nth-child(even) { background-color: #F8FAFC; }
tbody tr:hover            { background-color: #F1F5F9; }

tbody td {
  padding: 1.8mm 3mm;
  border-bottom: 1px solid #E2E8F0;
  color: #1A1A2E;
}

/* ─── Blockquote ─────────────────────────────────────────────────────────── */
blockquote {
  border-left: 4px solid #BE123C;
  margin: 4mm 0;
  padding: 2.5mm 5mm;
  background: #FFF1F2;
  color: #4A1124;
  font-style: italic;
  page-break-inside: avoid;
  break-inside: avoid;
}

/* ─── Code ───────────────────────────────────────────────────────────────── */
code {
  font-family: 'JetBrains Mono', 'Courier New', monospace;
  font-size: 9pt;
  background: #F1F5F9;
  padding: 0.4mm 1.6mm;
  border-radius: 2px;
  color: #0F172A;
}

pre {
  background: #0F172A;
  color: #E2E8F0;
  padding: 4mm 5mm;
  border-radius: 3px;
  font-family: 'JetBrains Mono', 'Courier New', monospace;
  font-size: 8.5pt;
  line-height: 1.5;
  overflow-x: auto;
  page-break-inside: avoid;
  break-inside: avoid;
  margin: 3mm 0 4mm;
}

pre code {
  background: transparent;
  padding: 0;
  color: inherit;
  font-size: inherit;
}

/* ─── Info / Warning Boxes ───────────────────────────────────────────────── */
.info-box {
  background: #F0FDF4;
  border: 1px solid #BBF7D0;
  border-left: 4px solid #064E3B;
  padding: 3mm 4mm;
  margin: 4mm 0;
  border-radius: 2px;
  page-break-inside: avoid;
  break-inside: avoid;
}

.warning-box {
  background: #FFF7ED;
  border: 1px solid #FED7AA;
  border-left: 4px solid #EA580C;
  padding: 3mm 4mm;
  margin: 4mm 0;
  border-radius: 2px;
  page-break-inside: avoid;
  break-inside: avoid;
}

/* ─── Divider ────────────────────────────────────────────────────────────── */
hr {
  border: none;
  border-top: 1px solid #E2E8F0;
  margin: 5mm 0;
}

/* ─── Print Adjustments ──────────────────────────────────────────────────── */
@media print {
  a { color: inherit; text-decoration: none; }
  .page-header, .page-footer, .watermark { display: block !important; }
  .no-print { display: none !important; }
}
`
