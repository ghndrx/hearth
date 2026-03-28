#!/usr/bin/env node
/**
 * WCAG Contrast Ratio Checker for Hearth
 * Audits color combinations against WCAG 2.1 AA requirements
 * 
 * Usage: node scripts/contrast-checker.js [--json] [--ci]
 */

// Convert hex to RGB
function hexToRgb(hex) {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
  return result ? {
    r: parseInt(result[1], 16),
    g: parseInt(result[2], 16),
    b: parseInt(result[3], 16)
  } : null;
}

// Convert rgba string to RGB (ignoring alpha for contrast calc)
function rgbaToRgb(rgba, bgHex = '#000000') {
  const match = rgba.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)(?:,\s*[\d.]+)?\)/);
  if (match) {
    return { r: parseInt(match[1]), g: parseInt(match[2]), b: parseInt(match[3]) };
  }
  return hexToRgb(bgHex);
}

// Calculate relative luminance (WCAG formula)
function getLuminance(rgb) {
  const [r, g, b] = [rgb.r, rgb.g, rgb.b].map(v => {
    v /= 255;
    return v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4);
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

// Calculate contrast ratio
function getContrastRatio(color1, color2) {
  const l1 = getLuminance(color1);
  const l2 = getLuminance(color2);
  const lighter = Math.max(l1, l2);
  const darker = Math.min(l1, l2);
  return (lighter + 0.05) / (darker + 0.05);
}

// Format ratio with 2 decimal places
function formatRatio(ratio) {
  return ratio.toFixed(2) + ':1';
}

// WCAG AA requirements
const WCAG_AA_NORMAL_TEXT = 4.5;  // Normal text (<18pt or <14pt bold)
const WCAG_AA_LARGE_TEXT = 3.0;   // Large text (≥18pt or ≥14pt bold)
const WCAG_AAA_NORMAL_TEXT = 7.0; // AAA level
const WCAG_AA_UI = 3.0;           // UI components and graphical objects

// Hearth Design Tokens (A11Y-003: Updated for WCAG 4.5:1 compliance)
const colors = {
  // From theme.css - Dark theme (default)
  dark: {
    backgrounds: {
      'bg-primary': '#313338',
      'bg-secondary': '#2b2d31',
      'bg-tertiary': '#1e1f22',
    },
    text: {
      'text-primary': '#f2f3f5',
      'text-secondary': '#b5bac1',
      'text-muted': '#949ba4',
      'text-link': '#00a8fc',
      'text-positive': '#2dc766',  // A11Y-003: Was #23a559
      'text-warning': '#f0b232',
      'text-danger': '#ff6b6f',    // A11Y-003: Was #f23f43
    },
    brand: {
      'brand-primary': '#8b94f7',  // A11Y-003: Was #5865f2
      'brand-hover': '#8b94f7',    // A11Y-003: Same as primary, hover uses filter
    },
    status: {
      'status-online': '#2dc766',  // A11Y-003: Was #23a559
      'status-idle': '#f0b232',
      'status-dnd': '#ff6b6f',     // A11Y-003: Was #f23f43
      'status-offline': '#80848e',
    }
  },
  // Light theme
  light: {
    backgrounds: {
      'bg-primary': '#ffffff',
      'bg-secondary': '#f2f3f5',
      'bg-tertiary': '#e3e5e8',
    },
    text: {
      'text-primary': '#060607',
      'text-secondary': '#4e5058',
      'text-muted': '#5f6169',     // A11Y-003: Was #6d6f78
      'text-link': '#0059bb',      // A11Y-003: Was #006ce7
    },
    brand: {
      'brand-primary': '#4752c4',  // A11Y-003: Darkened for light backgrounds
      'brand-hover': '#3c46a8',
    }
  },
  // Midnight theme
  midnight: {
    backgrounds: {
      'bg-primary': '#000000',
      'bg-secondary': '#0a0a0a',
      'bg-tertiary': '#0f0f0f',
    }
  },
  // Tailwind hearth palette
  hearth: {
    50: '#fef7ee',
    100: '#fcecd7',
    200: '#f8d5ae',
    300: '#f3b87a',
    400: '#ed9344',
    500: '#e87620',
    600: '#d95e16',
    700: '#b44614',
    800: '#903918',
    900: '#743116',
    950: '#3f170a'
  },
  // Tailwind dark palette
  tailwindDark: {
    50: '#f6f6f7',
    100: '#e1e2e6',
    200: '#c3c5cc',
    300: '#9ea1ab',
    400: '#797d8a',
    500: '#5f6370',
    600: '#4b4e59',
    700: '#3e404a',
    800: '#35373e',
    900: '#2f3136',
    950: '#1e1f22'
  }
};

// Define combinations to test
const testCases = [];

// Dark theme text on backgrounds
for (const [bgName, bgColor] of Object.entries(colors.dark.backgrounds)) {
  for (const [textName, textColor] of Object.entries(colors.dark.text)) {
    testCases.push({
      theme: 'dark',
      category: 'text',
      foreground: { name: textName, color: textColor },
      background: { name: bgName, color: bgColor },
      requirement: WCAG_AA_NORMAL_TEXT
    });
  }
  // Brand colors on backgrounds
  for (const [brandName, brandColor] of Object.entries(colors.dark.brand)) {
    testCases.push({
      theme: 'dark',
      category: 'brand',
      foreground: { name: brandName, color: brandColor },
      background: { name: bgName, color: bgColor },
      requirement: WCAG_AA_NORMAL_TEXT
    });
  }
  // Status colors on backgrounds
  for (const [statusName, statusColor] of Object.entries(colors.dark.status)) {
    testCases.push({
      theme: 'dark',
      category: 'status',
      foreground: { name: statusName, color: statusColor },
      background: { name: bgName, color: bgColor },
      requirement: WCAG_AA_UI
    });
  }
}

// Light theme text on backgrounds
for (const [bgName, bgColor] of Object.entries(colors.light.backgrounds)) {
  for (const [textName, textColor] of Object.entries(colors.light.text)) {
    testCases.push({
      theme: 'light',
      category: 'text',
      foreground: { name: textName, color: textColor },
      background: { name: bgName, color: bgColor },
      requirement: WCAG_AA_NORMAL_TEXT
    });
  }
  // Light theme brand colors
  for (const [brandName, brandColor] of Object.entries(colors.light.brand)) {
    testCases.push({
      theme: 'light',
      category: 'brand',
      foreground: { name: brandName, color: brandColor },
      background: { name: bgName, color: bgColor },
      requirement: WCAG_AA_NORMAL_TEXT
    });
  }
}

// Midnight theme - inherits dark theme text colors
for (const [bgName, bgColor] of Object.entries(colors.midnight.backgrounds)) {
  for (const [textName, textColor] of Object.entries(colors.dark.text)) {
    testCases.push({
      theme: 'midnight',
      category: 'text',
      foreground: { name: textName, color: textColor },
      background: { name: bgName, color: bgColor },
      requirement: WCAG_AA_NORMAL_TEXT
    });
  }
}

// Button combinations (hearth colors) - A11Y-003: Black text on orange backgrounds
testCases.push({
  theme: 'component',
  category: 'button',
  foreground: { name: 'black-text', color: '#000000' },
  background: { name: 'hearth-500', color: colors.hearth['500'] },
  requirement: WCAG_AA_NORMAL_TEXT
});
// Hover state: hearth-600 (#d95e16) with brightness(110%) = ~#ee6718 (5.93:1 contrast)
testCases.push({
  theme: 'component',
  category: 'button-hover',
  foreground: { name: 'black-text', color: '#000000' },
  background: { name: 'hearth-600-bright (computed)', color: '#ee6718' },
  requirement: WCAG_AA_NORMAL_TEXT
});
testCases.push({
  theme: 'component',
  category: 'button',
  foreground: { name: 'gray-100', color: '#f3f4f6' },
  background: { name: 'dark-700', color: colors.tailwindDark['700'] },
  requirement: WCAG_AA_NORMAL_TEXT
});
testCases.push({
  theme: 'component',
  category: 'button',
  foreground: { name: 'gray-100', color: '#f3f4f6' },
  background: { name: 'dark-600', color: colors.tailwindDark['600'] },
  requirement: WCAG_AA_NORMAL_TEXT
});

// Run tests
const results = {
  passing: [],
  failing: [],
  warnings: [] // Passes AA but fails AAA
};

for (const test of testCases) {
  const fgRgb = hexToRgb(test.foreground.color);
  const bgRgb = hexToRgb(test.background.color);
  
  if (!fgRgb || !bgRgb) {
    console.error(`Invalid color: ${test.foreground.color} or ${test.background.color}`);
    continue;
  }
  
  const ratio = getContrastRatio(fgRgb, bgRgb);
  const passes = ratio >= test.requirement;
  const passesAAA = ratio >= WCAG_AAA_NORMAL_TEXT;
  
  const result = {
    ...test,
    ratio: ratio,
    ratioFormatted: formatRatio(ratio),
    passes: passes,
    passesAAA: passesAAA
  };
  
  if (passes) {
    if (!passesAAA && test.category === 'text') {
      results.warnings.push(result);
    }
    results.passing.push(result);
  } else {
    results.failing.push(result);
  }
}

// Output
const args = process.argv.slice(2);
const jsonOutput = args.includes('--json');
const ciMode = args.includes('--ci');

if (jsonOutput) {
  console.log(JSON.stringify(results, null, 2));
} else {
  console.log('\n╔══════════════════════════════════════════════════════════════════╗');
  console.log('║          HEARTH COLOR CONTRAST AUDIT - WCAG 2.1 AA               ║');
  console.log('╚══════════════════════════════════════════════════════════════════╝\n');
  
  console.log(`Total combinations tested: ${testCases.length}`);
  console.log(`✅ Passing: ${results.passing.length}`);
  console.log(`❌ Failing: ${results.failing.length}`);
  console.log(`⚠️  Warnings (AA pass, AAA fail): ${results.warnings.length}\n`);
  
  if (results.failing.length > 0) {
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('❌ FAILING COMBINATIONS (must fix for WCAG AA compliance)');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n');
    
    for (const fail of results.failing) {
      console.log(`Theme: ${fail.theme} | Category: ${fail.category}`);
      console.log(`  ${fail.foreground.name} (${fail.foreground.color}) on ${fail.background.name} (${fail.background.color})`);
      console.log(`  Ratio: ${fail.ratioFormatted} (required: ${fail.requirement}:1)`);
      console.log(`  Gap: ${(fail.requirement - fail.ratio).toFixed(2)}\n`);
    }
  }
  
  if (results.warnings.length > 0 && !ciMode) {
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('⚠️  WARNINGS (passes AA, fails AAA - consider improving)');
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n');
    
    for (const warn of results.warnings) {
      console.log(`  ${warn.foreground.name} on ${warn.background.name}: ${warn.ratioFormatted}`);
    }
    console.log('');
  }
  
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  console.log('✅ PASSING COMBINATIONS');
  console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n');
  
  // Group by theme for readability
  const byTheme = {};
  for (const pass of results.passing) {
    if (!byTheme[pass.theme]) byTheme[pass.theme] = [];
    byTheme[pass.theme].push(pass);
  }
  
  for (const [theme, passes] of Object.entries(byTheme)) {
    console.log(`[${theme.toUpperCase()}]`);
    for (const p of passes) {
      const aaa = p.passesAAA ? '(AAA)' : '';
      console.log(`  ✓ ${p.foreground.name} on ${p.background.name}: ${p.ratioFormatted} ${aaa}`);
    }
    console.log('');
  }
}

// Exit with error code if failing (for CI)
if (ciMode && results.failing.length > 0) {
  process.exit(1);
}
