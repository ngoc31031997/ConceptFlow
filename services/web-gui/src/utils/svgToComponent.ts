/**
 * CR-044 — turn an uploaded SVG file into a library drawing's TSX, in the
 * browser, so it lands in the same editor as hand-written code: the Creator
 * sees the code, previews it, and the renderer's style check warns about what
 * does not fit the channel (off-palette colours, outlines) before saving.
 *
 * The result is wrapped in <Figure> with the SVG's own viewBox, so it is
 * placed by x/y/size like every other drawing. It does not move: motion is
 * added by hand or by the AI redraw.
 */

const DROP_TAGS = new Set(["script", "metadata", "title", "desc", "style", "sodipodi:namedview"]);
const KEEP_AS_IS = new Set(["viewBox"]);

function camel(name: string): string {
  if (name === "class") return "className";
  if (name === "xlink:href") return "href";
  if (name.startsWith("xmlns") || name.includes(":")) return "";
  if (name.startsWith("data-") || name.startsWith("aria-")) return name;
  return name.replace(/-([a-z])/g, (_, c: string) => c.toUpperCase());
}

function styleObject(css: string): string {
  const pairs = css
    .split(";")
    .map((d) => d.split(":").map((s) => s.trim()))
    .filter(([k, v]) => k && v)
    .map(([k, v]) => `${camel(k)}: ${JSON.stringify(v)}`);
  return `{{${pairs.join(", ")}}}`;
}

function jsxAttr(value: string): string {
  return /^-?\d+(\.\d+)?$/.test(value) ? `{${value}}` : JSON.stringify(value);
}

function toJsx(el: Element, depth: number): string {
  const tag = el.tagName;
  if (DROP_TAGS.has(tag) || tag.includes(":")) return "";
  const pad = "  ".repeat(depth);
  const attrs: string[] = [];
  for (const a of Array.from(el.attributes)) {
    const name = KEEP_AS_IS.has(a.name) ? a.name : camel(a.name);
    if (!name || name === "id" && !el.closest("defs") && tag !== "clipPath") continue;
    attrs.push(name === "style" ? `style=${styleObject(a.value)}` : `${name}=${jsxAttr(a.value)}`);
  }
  const children = Array.from(el.children).map((c) => toJsx(c, depth + 1)).filter(Boolean);
  const open = `${pad}<${tag}${attrs.length ? " " + attrs.join(" ") : ""}`;
  if (children.length === 0) return `${open} />`;
  return `${open}>\n${children.join("\n")}\n${pad}</${tag}>`;
}

/** "school bus.svg" → "SchoolBus". Falls back to "Uploaded" for names without letters. */
export function componentNameFromFile(fileName: string): string {
  const base = fileName.replace(/\.svg$/i, "").normalize("NFD").replace(/[̀-ͯ]/g, "").replace(/đ/gi, "d");
  const words = base.split(/[^A-Za-z0-9]+/).filter(Boolean);
  const name = words.map((w) => w[0].toUpperCase() + w.slice(1)).join("").replace(/^\d+/, "");
  return name.length >= 2 ? name.slice(0, 41) : "Uploaded";
}

export interface ConvertedSvg {
  name: string;
  code: string;
  width: number;
  height: number;
}

export function svgToComponent(svgText: string, name: string): ConvertedSvg {
  const doc = new DOMParser().parseFromString(svgText, "image/svg+xml");
  const svg = doc.documentElement;
  if (svg.tagName !== "svg" || doc.getElementsByTagName("parsererror").length > 0) {
    throw new Error("File không phải SVG hợp lệ.");
  }
  const vb = (svg.getAttribute("viewBox") ?? "").split(/[\s,]+/).map(Number);
  let [minX, minY, width, height] = vb.length === 4 && vb.every(Number.isFinite) ? vb : [0, 0, NaN, NaN];
  if (!Number.isFinite(width)) {
    width = parseFloat(svg.getAttribute("width") ?? "") || 200;
    height = parseFloat(svg.getAttribute("height") ?? "") || 200;
    minX = 0;
    minY = 0;
  }
  const body = Array.from(svg.children).map((c) => toJsx(c, 3)).filter(Boolean).join("\n");
  if (!body) throw new Error("SVG không có hình nào để dùng.");
  const shift = minX || minY ? `\n    <g transform="translate(${-minX} ${-minY})">\n${body}\n    </g>` : `\n${body}`;
  const code = `import React from 'react';
import {Figure, type FigureProps} from './conceptflow-mini/illustration';

// Tải lên từ file SVG. Hộp ${width} x ${height}. Hình đứng yên — thêm chuyển động bằng tay hoặc "Vẽ lại bằng AI".
export function ${name}({...fig}: FigureProps) {
  return (
    <Figure {...fig} size={fig.size ?? 280} vw={${width}} vh={${height}}>${shift}
    </Figure>
  );
}
`;
  return { name, code, width, height };
}
