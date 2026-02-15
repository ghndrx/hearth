#!/usr/bin/env node
/**
 * A11Y-003: WCAG Color Contrast Audit Script
 * Checks color combinations against WCAG 2.1 AA requirements:
 * - Normal text: 4.5:1
 * - Large text (18pt+): 3:1
 * - UI components: 3:1
 */

// Convert hex to RGB
function hexToRgb(hex) {
  hex = hex.replace('#', '');
  if (hex.length === 3) {
    hex = hex.split('').map(c => c + c).join('');
  }
  const num = parseInt(hex, 16);
  return {
    r: (num >> 16) & 255,
    g: (num >> 8) & 255,
    b: num & 255
  };
}

// Calculate relative luminance per WCAG 2.1
function getLuminance(rgb) {
  const [r, g, b] = [rgb.r, rgb.g, rgb.b].map(c => {
    c = c / 255;
    return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

// Calculate contrast ratio
function getContrastRatio(color1, color2) {
  const lum1 = getLuminance(hexToRgb(color1));
  const lum2 = getLuminance(hexToRgb(color2));
  const lighter = Math.max(lum1, lum2);
  const darker = Math.min(lum1, lum2);
  return (lighter + 0.05) / (darker + 0.05);
}

// Color definitions from theme
const colors = {
  // Backgrounds (dark theme)
  bgPrimary: '#313338',
  bgSecondary: '#2b2d31',
  bgTertiary: '#1e1f22',
  
  // Backgrounds (light theme)
  bgPrimaryLight: '#ffffff',
  bgSecondaryLight: '#f2f3f5',
  bgTertiaryLight: '#e3e5e8',
  
  // Backgrounds (midnight theme)
  bgPrimaryMidnight: '#000000',
  bgSecondaryMidnight: '#0a0a0a',
  bgTertiaryMidnight: '#0f0f0f',
  
  // Text (dark theme)
  textPrimary: '#f2f3f5',
  textSecondary: '#b5bac1',
  textMuted: '#949ba4',
  textLink: '#00a8fc',
  textPositive: '#2dc766',
  textWarning: '#f0b232',
  textDanger: '#ff6b6f',
  
  // Text (light theme)
  textPrimaryLight: '#060607',
  textSecondaryLight: '#4e5058',
  textMutedLight: '#5f6169',
  textLinkLight: '#0059bb',
  
  // Brand (A11Y-003 updated)
  brandPrimary: '#8b94f7',
  brandHover: '#8b94f7',
  brandPrimaryLight: '#4752c4',
  
  // Hearth brand (Tailwind) - A11Y-003 updated
  hearth500: '#e87620',
  hearth600: '#e06518',  // Brightened for 4.74:1 contrast
  hearth700: '#b44614',
  
  // Dark theme from Tailwind
  dark300: '#9ea1ab',  // A11Y-003: Used for channel text
  dark600: '#4b4e59',
  dark700: '#3e404a',
  dark800: '#35373e',
  dark900: '#2f3136',
  dark950: '#1e1f22',
  
  // Gray shades for text
  gray100: '#e1e2e6',
  gray400: '#797d8a',
  gray500: '#5f6370',
  
  // Status colors
  statusOnline: '#2dc766',
  statusIdle: '#f0b232',
  statusDnd: '#ff6b6f',
  statusOffline: '#80848e'
};

// Define checks: [foreground, background, description, requirement]
const checks = [
  // Dark theme text on backgrounds
  ['textPrimary', 'bgPrimary', 'Primary text on primary bg (dark)', 4.5],
  ['textPrimary', 'bgSecondary', 'Primary text on secondary bg (dark)', 4.5],
  ['textPrimary', 'bgTertiary', 'Primary text on tertiary bg (dark)', 4.5],
  ['textSecondary', 'bgPrimary', 'Secondary text on primary bg (dark)', 4.5],
  ['textSecondary', 'bgSecondary', 'Secondary text on secondary bg (dark)', 4.5],
  ['textMuted', 'bgPrimary', 'Muted text on primary bg (dark)', 4.5],
  ['textMuted', 'bgSecondary', 'Muted text on secondary bg (dark)', 4.5],
  
  // Links
  ['textLink', 'bgPrimary', 'Link on primary bg (dark)', 4.5],
  ['textLink', 'bgSecondary', 'Link on secondary bg (dark)', 4.5],
  ['textLinkLight', 'bgPrimaryLight', 'Link on primary bg (light)', 4.5],
  
  // Status/semantic colors
  ['textPositive', 'bgPrimary', 'Positive/success text on primary bg (dark)', 4.5],
  ['textWarning', 'bgPrimary', 'Warning text on primary bg (dark)', 4.5],
  ['textDanger', 'bgPrimary', 'Danger text on primary bg (dark)', 4.5],
  ['statusOnline', 'bgSecondary', 'Online status on secondary bg (dark)', 3],
  ['statusIdle', 'bgSecondary', 'Idle status on secondary bg (dark)', 3],
  ['statusDnd', 'bgSecondary', 'DND status on secondary bg (dark)', 3],
  ['statusOffline', 'bgSecondary', 'Offline status on secondary bg (dark)', 3],
  
  // Brand colors
  ['brandPrimary', 'bgPrimary', 'Brand primary on primary bg (dark)', 4.5],
  ['brandPrimary', 'bgSecondary', 'Brand primary on secondary bg (dark)', 4.5],
  ['brandPrimaryLight', 'bgPrimaryLight', 'Brand primary on white (light)', 4.5],
  
  // Light theme
  ['textPrimaryLight', 'bgPrimaryLight', 'Primary text on white (light)', 4.5],
  ['textPrimaryLight', 'bgSecondaryLight', 'Primary text on secondary bg (light)', 4.5],
  ['textSecondaryLight', 'bgPrimaryLight', 'Secondary text on white (light)', 4.5],
  ['textMutedLight', 'bgPrimaryLight', 'Muted text on white (light)', 4.5],
  
  // Midnight theme
  ['textPrimary', 'bgPrimaryMidnight', 'Primary text on black (midnight)', 4.5],
  ['textSecondary', 'bgPrimaryMidnight', 'Secondary text on black (midnight)', 4.5],
  ['textMuted', 'bgPrimaryMidnight', 'Muted text on black (midnight)', 4.5],
  
  // Button text contrast (Hearth brand buttons)
  ['dark950', 'hearth500', 'Dark text on hearth-500 button', 4.5],
  ['dark950', 'hearth600', 'Dark text on hearth-600 button (hover)', 4.5],
  ['gray100', 'dark700', 'Light text on dark-700 button', 4.5],
  
  // Interactive elements (channel items, sidebar) - A11Y-003: now using dark-300
  ['dark300', 'dark900', 'Dark-300 channel text on dark-900 (was gray-400)', 4.5],
  ['gray100', 'dark600', 'Active item text on dark-600', 4.5],
];

console.log('='.repeat(80));
console.log('WCAG 2.1 AA Color Contrast Audit - Hearth Design System');
console.log('='.repeat(80));
console.log('');

let passing = 0;
let failing = 0;
const issues = [];

checks.forEach(([fg, bg, desc, req]) => {
  const fgColor = colors[fg];
  const bgColor = colors[bg];
  const ratio = getContrastRatio(fgColor, bgColor);
  const pass = ratio >= req;
  
  if (pass) {
    passing++;
    console.log(`✅ PASS ${ratio.toFixed(2)}:1 (need ${req}:1) - ${desc}`);
    console.log(`   ${fgColor} on ${bgColor}`);
  } else {
    failing++;
    console.log(`❌ FAIL ${ratio.toFixed(2)}:1 (need ${req}:1) - ${desc}`);
    console.log(`   ${fgColor} on ${bgColor}`);
    issues.push({ fg, bg, fgColor, bgColor, desc, ratio, req });
  }
  console.log('');
});

console.log('='.repeat(80));
console.log(`SUMMARY: ${passing} passing, ${failing} failing`);
console.log('='.repeat(80));

if (issues.length > 0) {
  console.log('\n⚠️  ISSUES REQUIRING FIXES:\n');
  issues.forEach((issue, i) => {
    console.log(`${i + 1}. ${issue.desc}`);
    console.log(`   Current: ${issue.fgColor} on ${issue.bgColor} = ${issue.ratio.toFixed(2)}:1`);
    console.log(`   Required: ${issue.req}:1`);
    console.log('');
  });
}

// Export for documentation
console.log('\n📋 AUDIT COMPLETE');
