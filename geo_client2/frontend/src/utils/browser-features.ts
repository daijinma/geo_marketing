/**
 * Browser feature detection utilities
 * Used for fallback rendering when certain JS engine features are unavailable
 */

/**
 * Detect if the current JS engine supports named capture groups in RegExp.
 * Named capture groups: /(?<name>pattern)/
 * 
 * Older JS engines (pre-ES2018) will throw a SyntaxError when parsing patterns with named groups.
 * This detection uses a try-catch to test regex compilation.
 */
export function supportsNamedCaptureGroups(): boolean {
  try {
    // Try to compile a regex with a named capture group
    // This will throw a SyntaxError in unsupported engines
    new RegExp('(?<test>x)');
    return true;
  } catch {
    return false;
  }
}

/**
 * Cached feature detection result (computed once per session)
 */
let cachedNamedGroupsSupport: boolean | null = null;

/**
 * Get cached or compute named capture group support
 */
export function hasNamedCaptureGroups(): boolean {
  if (cachedNamedGroupsSupport === null) {
    cachedNamedGroupsSupport = supportsNamedCaptureGroups();
  }
  return cachedNamedGroupsSupport;
}
