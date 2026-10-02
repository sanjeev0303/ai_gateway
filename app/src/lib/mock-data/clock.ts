/**
 * Single clock for the whole fixture set. Every relative timestamp derives
 * from this, so the dataset renders identically on server and client.
 *
 * This lives in its own leaf module (not in organization.ts) so that
 * organization.ts can derive its totals from cost.ts without creating an
 * import cycle through the modules cost.ts depends on.
 */
export const NOW = new Date("2026-09-29T12:00:00Z");
