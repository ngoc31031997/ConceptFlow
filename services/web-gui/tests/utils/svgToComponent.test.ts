import { describe, it, expect } from "vitest";
import { componentNameFromFile, svgToComponent } from "../../src/utils/svgToComponent";

const SVG = `<?xml version="1.0"?>
<svg xmlns="http://www.w3.org/2000/svg" xmlns:inkscape="x" viewBox="0 0 120 80" width="120" height="80">
  <title>bus</title>
  <metadata>junk</metadata>
  <rect x="10" y="10" width="100" height="50" rx="12" fill="#FFC72C" stroke-width="2" class="body"/>
  <circle cx="30" cy="65" r="10" style="fill: #2B2140; fill-opacity: 0.9"/>
  <script>alert(1)</script>
</svg>`;

describe("svgToComponent (CR-044)", () => {
  it("wraps the shapes in Figure with the SVG's own box, in JSX spelling", () => {
    const out = svgToComponent(SVG, "SchoolBus");
    expect(out.width).toBe(120);
    expect(out.height).toBe(80);
    expect(out.code).toContain("export function SchoolBus({...fig}: FigureProps)");
    expect(out.code).toContain("vw={120} vh={80}");
    expect(out.code).toContain('<rect x={10} y={10} width={100} height={50} rx={12} fill="#FFC72C" strokeWidth={2} className="body" />');
    expect(out.code).toContain('style={{fill: "#2B2140", fillOpacity: "0.9"}}');
    expect(out.code).not.toMatch(/script|metadata|<title|inkscape|xmlns/);
  });

  it("moves a viewBox that does not start at 0,0", () => {
    const out = svgToComponent(SVG.replace('viewBox="0 0 120 80"', 'viewBox="-10 -5 120 80"'), "Bus");
    expect(out.code).toContain('<g transform="translate(10 5)">');
  });

  it("refuses what is not an SVG", () => {
    expect(() => svgToComponent("<html></html>", "X")).toThrow("SVG");
  });

  it("derives a PascalCase name from the file name, Vietnamese included", () => {
    expect(componentNameFromFile("xe buýt trường học.svg")).toBe("XeBuytTruongHoc");
    expect(componentNameFromFile("school-bus_v2.SVG")).toBe("SchoolBusV2");
    expect(componentNameFromFile("123.svg")).toBe("Uploaded");
  });
});
