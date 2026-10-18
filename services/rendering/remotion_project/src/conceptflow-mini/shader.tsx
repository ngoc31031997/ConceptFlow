/**
 * conceptflow-mini/shader — full-layer WebGL2 backdrops: sky, nebula, water,
 * caustics, fog, god rays, tissue and aurora.
 *
 * A drawing never writes GLSL: it picks one of the shaders below and hands in
 * palette colours. The picture depends only on the frame number (time =
 * frame / fps), so a frame renders the same every time. No WebGL2 context is
 * an error, not a silent blank.
 */
import React from 'react';
import {useCurrentFrame, useVideoConfig} from 'remotion';
import {parseHex} from './film';
import {useLayerBox} from './scene';

export type ShaderName = 'sky' | 'nebula' | 'water' | 'caustics' | 'fog' | 'godrays' | 'tissue' | 'aurora';

const HEADER = `#version 300 es
precision highp float;
uniform vec2 u_res;
uniform float u_time;
uniform float u_seed;
uniform float u_intensity;
uniform vec2 u_pos;
uniform vec3 u_c0;
uniform vec3 u_c1;
uniform vec3 u_c2;
out vec4 o;
float hash(vec2 p) {
  p = fract(p * vec2(123.34, 456.21) + u_seed);
  p += dot(p, p + 45.32);
  return fract(p.x * p.y);
}
float vnoise(vec2 p) {
  vec2 i = floor(p);
  vec2 f = fract(p);
  f = f * f * (3.0 - 2.0 * f);
  return mix(mix(hash(i), hash(i + vec2(1, 0)), f.x), mix(hash(i + vec2(0, 1)), hash(i + vec2(1, 1)), f.x), f.y);
}
float fbm(vec2 p) {
  float v = 0.0;
  float a = 0.5;
  for (int i = 0; i < 5; i++) {
    v += a * vnoise(p);
    p = p * 2.03 + 17.0;
    a *= 0.5;
  }
  return v;
}
void main() {
  vec2 uv = gl_FragCoord.xy / u_res;
  uv.y = 1.0 - uv.y;
  float aspect = u_res.x / u_res.y;
  vec2 p = vec2(uv.x * aspect, uv.y);
  float t = u_time;
  vec3 col = vec3(0.0);
  float alpha = 1.0;
`;

const FOOTER = `
  o = vec4(col * alpha, alpha);
}
`;

/** Fragment bodies; each sets `col` and, when translucent, `alpha`. */
const BODIES: Record<ShaderName, string> = {
  sky: `
  col = mix(u_c0, u_c1, smoothstep(0.0, 0.9, uv.y));
  float cl = smoothstep(0.45, 0.8, fbm(p * 2.2 + vec2(t * 0.02, 0.0)));
  col = mix(col, u_c2, cl * 0.5 * u_intensity * smoothstep(0.1, 0.6, uv.y));
`,
  nebula: `
  vec2 q = p * 1.6;
  vec2 w = vec2(fbm(q + t * 0.01), fbm(q + 5.2 - t * 0.01));
  float n = fbm(q + 2.5 * w);
  col = mix(u_c0, u_c1, smoothstep(0.2, 0.75, n));
  col = mix(col, u_c2, smoothstep(0.55, 0.95, fbm(q * 1.7 - w)) * u_intensity);
  float s = step(0.9975, hash(floor(gl_FragCoord.xy / 2.0)));
  col += vec3(s) * (0.6 + 0.4 * sin(t * 2.0 + hash(floor(gl_FragCoord.xy / 2.0)) * 20.0));
`,
  water: `
  col = mix(u_c0, u_c1, uv.y);
  float r = fbm(vec2(p.x * 5.0, p.y * 14.0 + t * 0.25));
  col = mix(col, u_c2, smoothstep(0.55, 0.85, r) * 0.45 * u_intensity);
`,
  caustics: `
  col = mix(u_c0, u_c1, uv.y);
  float a = abs(vnoise(p * 6.0 + vec2(t * 0.12, t * 0.08)) * 2.0 - 1.0);
  float b = abs(vnoise(p * 6.0 * 1.7 - vec2(t * 0.1, 0.0)) * 2.0 - 1.0);
  float c = pow(1.0 - min(a, b), 8.0);
  col += u_c2 * c * 0.8 * u_intensity;
`,
  fog: `
  float f = fbm(p * 2.0 + vec2(t * 0.015, 0.0));
  col = u_c0;
  alpha = clamp(f * 1.4 - 0.25, 0.0, 1.0) * u_intensity * mix(1.0, 0.4, uv.y);
`,
  godrays: `
  vec2 d = uv - u_pos;
  float ang = atan(d.y, d.x * aspect);
  float rays = pow(max(0.0, sin(ang * 14.0 + fbm(vec2(ang * 3.0, t * 0.05)) * 6.0)), 3.0);
  float fall = exp(-length(vec2(d.x * aspect, d.y)) * 1.6);
  col = u_c0;
  alpha = rays * fall * u_intensity;
`,
  tissue: `
  float r = length((uv - 0.5) * vec2(aspect, 1.0)) / (0.5 * aspect);
  col = mix(u_c0, u_c1, smoothstep(0.0, 1.0, r));
  float v = 1.0 - abs(fbm(p * 5.0 + vec2(0.0, t * 0.02)) * 2.0 - 1.0);
  col = mix(col, u_c2, pow(v, 6.0) * 0.55 * u_intensity);
`,
  aurora: `
  col = u_c0 * 0.0;
  float band = 0.0;
  for (int i = 0; i < 3; i++) {
    float fi = float(i);
    float y = 0.28 + fi * 0.12 + 0.07 * sin(p.x * (2.0 + fi) + t * (0.15 + fi * 0.05) + fbm(vec2(p.x * 2.0, fi)) * 4.0);
    band += smoothstep(0.16, 0.0, abs(uv.y - y)) * (0.5 + 0.5 * fbm(vec2(p.x * 8.0, uv.y * 2.0 + t * 0.1 + fi)));
  }
  col = mix(u_c0, u_c1, smoothstep(0.1, 0.9, uv.x)) + u_c2 * band * 0.35;
  alpha = clamp(band, 0.0, 1.0) * u_intensity;
`,
};

/** Full GLSL source of a shader. */
export function shaderSource(name: ShaderName): string {
  const body = BODIES[name];
  if (!body) throw new Error(`Shader "${name}" không có (có: ${Object.keys(BODIES).join(', ')})`);
  return HEADER + body + FOOTER;
}

/** Names of the shaders this kit offers. */
export const SHADER_NAMES = Object.keys(BODIES) as ShaderName[];

const VERTEX = `#version 300 es
in vec2 a;
void main() { gl_Position = vec4(a, 0.0, 1.0); }
`;

function compile(gl: WebGL2RenderingContext, type: number, source: string, label: string): WebGLShader {
  const shader = gl.createShader(type);
  if (!shader) throw new Error(`Không tạo được shader ${label}`);
  gl.shaderSource(shader, source);
  gl.compileShader(shader);
  if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
    throw new Error(`Shader ${label} lỗi: ${gl.getShaderInfoLog(shader)}`);
  }
  return shader;
}

function rgb(hex: string): [number, number, number] {
  const [r, g, b] = parseHex(hex);
  return [r / 255, g / 255, b / 255];
}

export interface ShaderLayerProps {
  shader: ShaderName;
  /** Two or three palette colours; the shader names what each one is for. */
  colors: [string, string] | [string, string, string];
  /** 0..1, how strong the shader's feature is. */
  intensity?: number;
  /** Source position for `godrays`, 0..1 on both axes. */
  pos?: [number, number];
  seed?: number;
}

/** One shader drawn over the whole layer box. */
export function ShaderLayer({shader, colors, intensity = 1, pos = [0.7, 0.1], seed = 1}: ShaderLayerProps) {
  const box = useLayerBox();
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const ref = React.useRef<HTMLCanvasElement>(null);
  const width = Math.round(box.width);
  const height = Math.round(box.height);

  React.useLayoutEffect(() => {
    const canvas = ref.current;
    if (!canvas) return;
    const gl = canvas.getContext('webgl2', {premultipliedAlpha: true, preserveDrawingBuffer: true, alpha: true});
    if (!gl) throw new Error('Trình duyệt không có WebGL2: không vẽ được lớp shader');
    const program = gl.createProgram();
    if (!program) throw new Error('Không tạo được chương trình shader');
    gl.attachShader(program, compile(gl, gl.VERTEX_SHADER, VERTEX, 'vertex'));
    gl.attachShader(program, compile(gl, gl.FRAGMENT_SHADER, shaderSource(shader), shader));
    gl.linkProgram(program);
    if (!gl.getProgramParameter(program, gl.LINK_STATUS)) throw new Error(`Liên kết shader ${shader} lỗi: ${gl.getProgramInfoLog(program)}`);
    gl.useProgram(program);

    const buffer = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
    gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 1, -1, -1, 1, 1, 1]), gl.STATIC_DRAW);
    const loc = gl.getAttribLocation(program, 'a');
    gl.enableVertexAttribArray(loc);
    gl.vertexAttribPointer(loc, 2, gl.FLOAT, false, 0, 0);

    const set2 = (name: string, x: number, y: number) => gl.uniform2f(gl.getUniformLocation(program, name), x, y);
    const set1 = (name: string, x: number) => gl.uniform1f(gl.getUniformLocation(program, name), x);
    const set3 = (name: string, c: [number, number, number]) => gl.uniform3f(gl.getUniformLocation(program, name), c[0], c[1], c[2]);
    gl.viewport(0, 0, width, height);
    set2('u_res', width, height);
    set1('u_time', frame / fps);
    set1('u_seed', seed);
    set1('u_intensity', intensity);
    set2('u_pos', pos[0], pos[1]);
    set3('u_c0', rgb(colors[0]));
    set3('u_c1', rgb(colors[1]));
    set3('u_c2', rgb(colors[2] ?? colors[1]));
    gl.clearColor(0, 0, 0, 0);
    gl.clear(gl.COLOR_BUFFER_BIT);
    gl.drawArrays(gl.TRIANGLE_STRIP, 0, 4);
    gl.deleteProgram(program);
    gl.deleteBuffer(buffer);
  }, [shader, colors, intensity, pos, seed, frame, fps, width, height]);

  return (
    <canvas
      ref={ref}
      width={width}
      height={height}
      style={{position: 'absolute', left: box.left, top: box.top, width: box.width, height: box.height}}
    />
  );
}
