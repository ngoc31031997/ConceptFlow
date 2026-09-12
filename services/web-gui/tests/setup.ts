import "@testing-library/jest-dom/vitest";

// jsdom doesn't implement matchMedia; ThemeContext reads it to seed the
// initial theme from the OS preference, so every test that touches
// ThemeProvider needs this stub in place.
// jsdom doesn't implement the Blob URL APIs used by download-as-file features.
if (!URL.createObjectURL) {
  URL.createObjectURL = () => "blob:mock-url";
}
if (!URL.revokeObjectURL) {
  URL.revokeObjectURL = () => {};
}

if (!window.matchMedia) {
  window.matchMedia = (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }) as unknown as MediaQueryList;
}
