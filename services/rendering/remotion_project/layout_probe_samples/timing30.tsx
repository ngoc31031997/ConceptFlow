import React from 'react';
import {registerRoot, Composition, AbsoluteFill, interpolate, interpolateColors, spring, Easing, useCurrentFrame, useVideoConfig} from 'remotion';
import {calculateMetadataFromSegments, Segments} from './conceptflow-mini/segments';
import {Stage, SAFE_MARGIN, WIDTH, HEIGHT} from './conceptflow-mini/primitives';
import {LottieClip} from './conceptflow-mini/lottie';
import {Backdrop, Panel, Person, Tooth, Germ, OpenMouth, Toothbrush, Toothpaste, Drop, Shield, Heart, Candy, Lollipop, Soda, Donut, Apple, Clock, Table, Chair, Window, Plant, House, Tree, Sun, Cloud, Lightbulb, Coin, Book, Phone, Magnifier, Mark, Sparkle, Airplane, Bubble, Figure, Face, GroundShadow, useBlink, phaseOf, shadeOf, useSvgId, INK, SHADE, BLUSH, WHITE} from './conceptflow-mini/illustration';
import type {FigureProps, Mood, PersonPose} from './conceptflow-mini/illustration';

const PALETTE = {
  accent: '#F5B841', // thứ đang được chú ý
  muted: '#4A5670', // nền phụ
  ink: '#F2F7FF', // nhãn và chữ
};

const LAYOUT = {
  hero: {x: 960, y: 560, size: 460},
  tooth: {x: 620, y: 560, size: 420},
  label: {x: 1180, y: 520, w: 560},
  cat: {x: 1500, y: 600, size: 420},
};

const clamp = {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'} as const;

export const narrations: string[] = [
  "Bạn có biết vì sao răng bị sâu không?",
  "Bên ngoài răng có một lớp men rất cứng.",
  "Vi khuẩn bám trên răng và tiết ra axit.",
  "Đánh răng đều đặn giúp bảo vệ lớp men.",
  "Kẹo ngọt là món ăn yêu thích của vi khuẩn.",
  "Hãy chọn táo thay cho kẹo nhé.",
  "Bạn có biết vì sao răng bị sâu không?",
  "Bên ngoài răng có một lớp men rất cứng.",
  "Vi khuẩn bám trên răng và tiết ra axit.",
  "Đánh răng đều đặn giúp bảo vệ lớp men.",
  "Kẹo ngọt là món ăn yêu thích của vi khuẩn.",
  "Hãy chọn táo thay cho kẹo nhé.",
  "Bạn có biết vì sao răng bị sâu không?",
  "Bên ngoài răng có một lớp men rất cứng.",
  "Vi khuẩn bám trên răng và tiết ra axit.",
  "Đánh răng đều đặn giúp bảo vệ lớp men.",
  "Kẹo ngọt là món ăn yêu thích của vi khuẩn.",
  "Hãy chọn táo thay cho kẹo nhé.",
  "Bạn có biết vì sao răng bị sâu không?",
  "Bên ngoài răng có một lớp men rất cứng.",
  "Vi khuẩn bám trên răng và tiết ra axit.",
  "Đánh răng đều đặn giúp bảo vệ lớp men.",
  "Kẹo ngọt là món ăn yêu thích của vi khuẩn.",
  "Hãy chọn táo thay cho kẹo nhé.",
  "Bạn có biết vì sao răng bị sâu không?",
  "Bên ngoài răng có một lớp men rất cứng.",
  "Vi khuẩn bám trên răng và tiết ra axit.",
  "Đánh răng đều đặn giúp bảo vệ lớp men.",
  "Kẹo ngọt là món ăn yêu thích của vi khuẩn.",
  "Hãy chọn táo thay cho kẹo nhé.",
];

type ShotProps = {duration: number};

// Shot 1.1 — MÁY: toàn cảnh, đứng yên | HÌNH: bạn nhỏ vẫy tay, tiêu đề hiện phía trên
function Shot1_1({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const grow = spring({frame, fps, config: {damping: 200}});
  const titleIn = interpolate(frame, [duration * 0.1, duration * 0.3], [0, 1], clamp);
  const {x, y, size} = LAYOUT.hero;
  return (
    <AbsoluteFill>
      <Person x={x} y={y + 60} size={size} pose="wave" mood="happy" age="child" scale={grow} />
      <div style={{position: 'absolute', left: 460, top: 130, width: 1000, textAlign: 'center', fontSize: 60, fontWeight: 700, lineHeight: 1.2, color: PALETTE.ink, opacity: titleIn}}>
        Vì sao răng bị sâu?
      </div>
    </AbsoluteFill>
  );
}

// Shot 1.2 — MÁY: đẩy vào chậm | HÌNH: chiếc răng bên trái, nhãn bên phải
function Shot1_2({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const zoom = interpolate(frame, [0, duration * 0.8], [1, 1.1], {...clamp, easing: Easing.inOut(Easing.cubic)});
  const labelIn = interpolate(frame, [duration * 0.2, duration * 0.4], [0, 1], clamp);
  const {x, y, size} = LAYOUT.tooth;
  return (
    <AbsoluteFill>
      <div style={{position: 'absolute', inset: 0, transformOrigin: `${x}px ${y}px`, transform: `scale(${zoom})`}}>
        <Tooth x={x} y={y} size={size} decay={0.2} />
        <div style={{position: 'absolute', left: LAYOUT.label.x - 180, top: LAYOUT.label.y - 30, width: LAYOUT.label.w, fontSize: 48, fontWeight: 700, lineHeight: 1.25, color: PALETTE.ink, opacity: labelIn}}>
          Men răng
        </div>
      </div>
    </AbsoluteFill>
  );
}

// Shot 1.3 — MÁY: trung cảnh | HÌNH: vi khuẩn trên răng, mũi tên chỉ vào, mèo suy nghĩ bên phải
function Shot1_3({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const arrow = interpolate(frame, [duration * 0.2, duration * 0.5], [0, 1], clamp);
  const {x, y, size} = LAYOUT.tooth;
  return (
    <AbsoluteFill>
      <Tooth x={x} y={y} size={size} decay={0.5} />
      <Germ x={x + 60} y={y - 120} size={140} />
      <svg width={300} height={80} viewBox="0 0 300 80" style={{position: 'absolute', left: 860, top: 400, opacity: arrow}}>
        <line x1={290} y1={40} x2={20} y2={40} stroke={PALETTE.accent} strokeWidth={10} strokeLinecap="round" />
        <polygon points="0,40 36,16 36,64" fill={PALETTE.accent} />
      </svg>
      <LottieClip id="cat.thinking" x={LAYOUT.cat.x} y={LAYOUT.cat.y} size={LAYOUT.cat.size} />
    </AbsoluteFill>
  );
}

// Shot 1.4 — MÁY: đứng yên | HÌNH: khiên bảo vệ bên phải, hai dòng chữ xếp bên trái
function Shot1_4({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const first = interpolate(frame, [0, duration * 0.2], [0, 1], clamp);
  const second = interpolate(frame, [duration * 0.3, duration * 0.5], [0, 1], clamp);
  return (
    <AbsoluteFill>
      <Shield x={1380} y={540} size={440} color={PALETTE.accent} />
      <div style={{position: 'absolute', left: 200, top: 380, width: 720, fontSize: 52, fontWeight: 700, lineHeight: 1.25, color: PALETTE.ink, opacity: first}}>
        Đánh răng hai lần mỗi ngày
      </div>
      <div style={{position: 'absolute', left: 200, top: 560, width: 720, fontSize: 44, lineHeight: 1.25, color: PALETTE.ink, opacity: second}}>
        Kem có fluor giúp men chắc hơn
      </div>
    </AbsoluteFill>
  );
}

// Shot 1.5 — MÁY: toàn cảnh | HÌNH: căn phòng, bạn nhỏ cầm kẹo, bong bóng thoại
function Shot1_5({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const pop = spring({frame: frame - Math.round(duration * 0.3), fps, config: {damping: 12}});
  return (
    <AbsoluteFill>
      <Backdrop color="#FFE3A3" floor="#E8B96A" floorY={820} />
      <Person x={760} y={580} size={520} pose="stand" mood="happy" age="child" />
      <Candy x={960} y={640} size={120} />
      <Bubble x={1320} y={340} w={420} h={170} scale={pop} fontSize={40}>
        Ngọt quá!
      </Bubble>
    </AbsoluteFill>
  );
}

// Shot 1.6 — MÁY: đứng yên | HÌNH: màn hình chia đôi, trái kẹo, phải táo, nhãn dưới mỗi bên
function Shot1_6({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const show = interpolate(frame, [0, duration * 0.25], [0, 1], clamp);
  return (
    <AbsoluteFill>
      <Panel x={120} y={120} w={820} h={840} color={PALETTE.muted} radius={32} />
      <Panel x={980} y={120} w={820} h={840} color={PALETTE.muted} radius={32} />
      <Lollipop x={530} y={480} size={380} />
      <Apple x={1390} y={480} size={360} />
      <div style={{position: 'absolute', left: 230, top: 760, width: 600, textAlign: 'center', fontSize: 48, fontWeight: 700, color: PALETTE.ink, opacity: show}}>
        Kẹo dính răng
      </div>
      <div style={{position: 'absolute', left: 1090, top: 760, width: 600, textAlign: 'center', fontSize: 48, fontWeight: 700, color: PALETTE.ink, opacity: show}}>
        Táo làm sạch răng
      </div>
    </AbsoluteFill>
  );
}

// Shot 2.1 — MÁY: toàn cảnh, đứng yên | HÌNH: bạn nhỏ vẫy tay, tiêu đề hiện phía trên
function Shot2_1({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const grow = spring({frame, fps, config: {damping: 200}});
  const titleIn = interpolate(frame, [duration * 0.1, duration * 0.3], [0, 1], clamp);
  const {x, y, size} = LAYOUT.hero;
  return (
    <AbsoluteFill>
      <Person x={x} y={y + 60} size={size} pose="wave" mood="happy" age="child" scale={grow} />
      <div style={{position: 'absolute', left: 460, top: 130, width: 1000, textAlign: 'center', fontSize: 60, fontWeight: 700, lineHeight: 1.2, color: PALETTE.ink, opacity: titleIn}}>
        Vì sao răng bị sâu?
      </div>
    </AbsoluteFill>
  );
}

// Shot 2.2 — MÁY: đẩy vào chậm | HÌNH: chiếc răng bên trái, nhãn bên phải
function Shot2_2({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const zoom = interpolate(frame, [0, duration * 0.8], [1, 1.1], {...clamp, easing: Easing.inOut(Easing.cubic)});
  const labelIn = interpolate(frame, [duration * 0.2, duration * 0.4], [0, 1], clamp);
  const {x, y, size} = LAYOUT.tooth;
  return (
    <AbsoluteFill>
      <div style={{position: 'absolute', inset: 0, transformOrigin: `${x}px ${y}px`, transform: `scale(${zoom})`}}>
        <Tooth x={x} y={y} size={size} decay={0.2} />
        <div style={{position: 'absolute', left: LAYOUT.label.x - 180, top: LAYOUT.label.y - 30, width: LAYOUT.label.w, fontSize: 48, fontWeight: 700, lineHeight: 1.25, color: PALETTE.ink, opacity: labelIn}}>
          Men răng
        </div>
      </div>
    </AbsoluteFill>
  );
}

// Shot 2.3 — MÁY: trung cảnh | HÌNH: vi khuẩn trên răng, mũi tên chỉ vào, mèo suy nghĩ bên phải
function Shot2_3({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const arrow = interpolate(frame, [duration * 0.2, duration * 0.5], [0, 1], clamp);
  const {x, y, size} = LAYOUT.tooth;
  return (
    <AbsoluteFill>
      <Tooth x={x} y={y} size={size} decay={0.5} />
      <Germ x={x + 60} y={y - 120} size={140} />
      <svg width={300} height={80} viewBox="0 0 300 80" style={{position: 'absolute', left: 860, top: 400, opacity: arrow}}>
        <line x1={290} y1={40} x2={20} y2={40} stroke={PALETTE.accent} strokeWidth={10} strokeLinecap="round" />
        <polygon points="0,40 36,16 36,64" fill={PALETTE.accent} />
      </svg>
      <LottieClip id="cat.thinking" x={LAYOUT.cat.x} y={LAYOUT.cat.y} size={LAYOUT.cat.size} />
    </AbsoluteFill>
  );
}

// Shot 2.4 — MÁY: đứng yên | HÌNH: khiên bảo vệ bên phải, hai dòng chữ xếp bên trái
function Shot2_4({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const first = interpolate(frame, [0, duration * 0.2], [0, 1], clamp);
  const second = interpolate(frame, [duration * 0.3, duration * 0.5], [0, 1], clamp);
  return (
    <AbsoluteFill>
      <Shield x={1380} y={540} size={440} color={PALETTE.accent} />
      <div style={{position: 'absolute', left: 200, top: 380, width: 720, fontSize: 52, fontWeight: 700, lineHeight: 1.25, color: PALETTE.ink, opacity: first}}>
        Đánh răng hai lần mỗi ngày
      </div>
      <div style={{position: 'absolute', left: 200, top: 560, width: 720, fontSize: 44, lineHeight: 1.25, color: PALETTE.ink, opacity: second}}>
        Kem có fluor giúp men chắc hơn
      </div>
    </AbsoluteFill>
  );
}

// Shot 2.5 — MÁY: toàn cảnh | HÌNH: căn phòng, bạn nhỏ cầm kẹo, bong bóng thoại
function Shot2_5({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const pop = spring({frame: frame - Math.round(duration * 0.3), fps, config: {damping: 12}});
  return (
    <AbsoluteFill>
      <Backdrop color="#FFE3A3" floor="#E8B96A" floorY={820} />
      <Person x={760} y={580} size={520} pose="stand" mood="happy" age="child" />
      <Candy x={960} y={640} size={120} />
      <Bubble x={1320} y={340} w={420} h={170} scale={pop} fontSize={40}>
        Ngọt quá!
      </Bubble>
    </AbsoluteFill>
  );
}

// Shot 2.6 — MÁY: đứng yên | HÌNH: màn hình chia đôi, trái kẹo, phải táo, nhãn dưới mỗi bên
function Shot2_6({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const show = interpolate(frame, [0, duration * 0.25], [0, 1], clamp);
  return (
    <AbsoluteFill>
      <Panel x={120} y={120} w={820} h={840} color={PALETTE.muted} radius={32} />
      <Panel x={980} y={120} w={820} h={840} color={PALETTE.muted} radius={32} />
      <Lollipop x={530} y={480} size={380} />
      <Apple x={1390} y={480} size={360} />
      <div style={{position: 'absolute', left: 230, top: 760, width: 600, textAlign: 'center', fontSize: 48, fontWeight: 700, color: PALETTE.ink, opacity: show}}>
        Kẹo dính răng
      </div>
      <div style={{position: 'absolute', left: 1090, top: 760, width: 600, textAlign: 'center', fontSize: 48, fontWeight: 700, color: PALETTE.ink, opacity: show}}>
        Táo làm sạch răng
      </div>
    </AbsoluteFill>
  );
}

// Shot 3.1 — MÁY: toàn cảnh, đứng yên | HÌNH: bạn nhỏ vẫy tay, tiêu đề hiện phía trên
function Shot3_1({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const grow = spring({frame, fps, config: {damping: 200}});
  const titleIn = interpolate(frame, [duration * 0.1, duration * 0.3], [0, 1], clamp);
  const {x, y, size} = LAYOUT.hero;
  return (
    <AbsoluteFill>
      <Person x={x} y={y + 60} size={size} pose="wave" mood="happy" age="child" scale={grow} />
      <div style={{position: 'absolute', left: 460, top: 130, width: 1000, textAlign: 'center', fontSize: 60, fontWeight: 700, lineHeight: 1.2, color: PALETTE.ink, opacity: titleIn}}>
        Vì sao răng bị sâu?
      </div>
    </AbsoluteFill>
  );
}

// Shot 3.2 — MÁY: đẩy vào chậm | HÌNH: chiếc răng bên trái, nhãn bên phải
function Shot3_2({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const zoom = interpolate(frame, [0, duration * 0.8], [1, 1.1], {...clamp, easing: Easing.inOut(Easing.cubic)});
  const labelIn = interpolate(frame, [duration * 0.2, duration * 0.4], [0, 1], clamp);
  const {x, y, size} = LAYOUT.tooth;
  return (
    <AbsoluteFill>
      <div style={{position: 'absolute', inset: 0, transformOrigin: `${x}px ${y}px`, transform: `scale(${zoom})`}}>
        <Tooth x={x} y={y} size={size} decay={0.2} />
        <div style={{position: 'absolute', left: LAYOUT.label.x - 180, top: LAYOUT.label.y - 30, width: LAYOUT.label.w, fontSize: 48, fontWeight: 700, lineHeight: 1.25, color: PALETTE.ink, opacity: labelIn}}>
          Men răng
        </div>
      </div>
    </AbsoluteFill>
  );
}

// Shot 3.3 — MÁY: trung cảnh | HÌNH: vi khuẩn trên răng, mũi tên chỉ vào, mèo suy nghĩ bên phải
function Shot3_3({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const arrow = interpolate(frame, [duration * 0.2, duration * 0.5], [0, 1], clamp);
  const {x, y, size} = LAYOUT.tooth;
  return (
    <AbsoluteFill>
      <Tooth x={x} y={y} size={size} decay={0.5} />
      <Germ x={x + 60} y={y - 120} size={140} />
      <svg width={300} height={80} viewBox="0 0 300 80" style={{position: 'absolute', left: 860, top: 400, opacity: arrow}}>
        <line x1={290} y1={40} x2={20} y2={40} stroke={PALETTE.accent} strokeWidth={10} strokeLinecap="round" />
        <polygon points="0,40 36,16 36,64" fill={PALETTE.accent} />
      </svg>
      <LottieClip id="cat.thinking" x={LAYOUT.cat.x} y={LAYOUT.cat.y} size={LAYOUT.cat.size} />
    </AbsoluteFill>
  );
}

// Shot 3.4 — MÁY: đứng yên | HÌNH: khiên bảo vệ bên phải, hai dòng chữ xếp bên trái
function Shot3_4({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const first = interpolate(frame, [0, duration * 0.2], [0, 1], clamp);
  const second = interpolate(frame, [duration * 0.3, duration * 0.5], [0, 1], clamp);
  return (
    <AbsoluteFill>
      <Shield x={1380} y={540} size={440} color={PALETTE.accent} />
      <div style={{position: 'absolute', left: 200, top: 380, width: 720, fontSize: 52, fontWeight: 700, lineHeight: 1.25, color: PALETTE.ink, opacity: first}}>
        Đánh răng hai lần mỗi ngày
      </div>
      <div style={{position: 'absolute', left: 200, top: 560, width: 720, fontSize: 44, lineHeight: 1.25, color: PALETTE.ink, opacity: second}}>
        Kem có fluor giúp men chắc hơn
      </div>
    </AbsoluteFill>
  );
}

// Shot 3.5 — MÁY: toàn cảnh | HÌNH: căn phòng, bạn nhỏ cầm kẹo, bong bóng thoại
function Shot3_5({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const pop = spring({frame: frame - Math.round(duration * 0.3), fps, config: {damping: 12}});
  return (
    <AbsoluteFill>
      <Backdrop color="#FFE3A3" floor="#E8B96A" floorY={820} />
      <Person x={760} y={580} size={520} pose="stand" mood="happy" age="child" />
      <Candy x={960} y={640} size={120} />
      <Bubble x={1320} y={340} w={420} h={170} scale={pop} fontSize={40}>
        Ngọt quá!
      </Bubble>
    </AbsoluteFill>
  );
}

// Shot 3.6 — MÁY: đứng yên | HÌNH: màn hình chia đôi, trái kẹo, phải táo, nhãn dưới mỗi bên
function Shot3_6({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const show = interpolate(frame, [0, duration * 0.25], [0, 1], clamp);
  return (
    <AbsoluteFill>
      <Panel x={120} y={120} w={820} h={840} color={PALETTE.muted} radius={32} />
      <Panel x={980} y={120} w={820} h={840} color={PALETTE.muted} radius={32} />
      <Lollipop x={530} y={480} size={380} />
      <Apple x={1390} y={480} size={360} />
      <div style={{position: 'absolute', left: 230, top: 760, width: 600, textAlign: 'center', fontSize: 48, fontWeight: 700, color: PALETTE.ink, opacity: show}}>
        Kẹo dính răng
      </div>
      <div style={{position: 'absolute', left: 1090, top: 760, width: 600, textAlign: 'center', fontSize: 48, fontWeight: 700, color: PALETTE.ink, opacity: show}}>
        Táo làm sạch răng
      </div>
    </AbsoluteFill>
  );
}

// Shot 4.1 — MÁY: toàn cảnh, đứng yên | HÌNH: bạn nhỏ vẫy tay, tiêu đề hiện phía trên
function Shot4_1({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const grow = spring({frame, fps, config: {damping: 200}});
  const titleIn = interpolate(frame, [duration * 0.1, duration * 0.3], [0, 1], clamp);
  const {x, y, size} = LAYOUT.hero;
  return (
    <AbsoluteFill>
      <Person x={x} y={y + 60} size={size} pose="wave" mood="happy" age="child" scale={grow} />
      <div style={{position: 'absolute', left: 460, top: 130, width: 1000, textAlign: 'center', fontSize: 60, fontWeight: 700, lineHeight: 1.2, color: PALETTE.ink, opacity: titleIn}}>
        Vì sao răng bị sâu?
      </div>
    </AbsoluteFill>
  );
}

// Shot 4.2 — MÁY: đẩy vào chậm | HÌNH: chiếc răng bên trái, nhãn bên phải
function Shot4_2({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const zoom = interpolate(frame, [0, duration * 0.8], [1, 1.1], {...clamp, easing: Easing.inOut(Easing.cubic)});
  const labelIn = interpolate(frame, [duration * 0.2, duration * 0.4], [0, 1], clamp);
  const {x, y, size} = LAYOUT.tooth;
  return (
    <AbsoluteFill>
      <div style={{position: 'absolute', inset: 0, transformOrigin: `${x}px ${y}px`, transform: `scale(${zoom})`}}>
        <Tooth x={x} y={y} size={size} decay={0.2} />
        <div style={{position: 'absolute', left: LAYOUT.label.x - 180, top: LAYOUT.label.y - 30, width: LAYOUT.label.w, fontSize: 48, fontWeight: 700, lineHeight: 1.25, color: PALETTE.ink, opacity: labelIn}}>
          Men răng
        </div>
      </div>
    </AbsoluteFill>
  );
}

// Shot 4.3 — MÁY: trung cảnh | HÌNH: vi khuẩn trên răng, mũi tên chỉ vào, mèo suy nghĩ bên phải
function Shot4_3({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const arrow = interpolate(frame, [duration * 0.2, duration * 0.5], [0, 1], clamp);
  const {x, y, size} = LAYOUT.tooth;
  return (
    <AbsoluteFill>
      <Tooth x={x} y={y} size={size} decay={0.5} />
      <Germ x={x + 60} y={y - 120} size={140} />
      <svg width={300} height={80} viewBox="0 0 300 80" style={{position: 'absolute', left: 860, top: 400, opacity: arrow}}>
        <line x1={290} y1={40} x2={20} y2={40} stroke={PALETTE.accent} strokeWidth={10} strokeLinecap="round" />
        <polygon points="0,40 36,16 36,64" fill={PALETTE.accent} />
      </svg>
      <LottieClip id="cat.thinking" x={LAYOUT.cat.x} y={LAYOUT.cat.y} size={LAYOUT.cat.size} />
    </AbsoluteFill>
  );
}

// Shot 4.4 — MÁY: đứng yên | HÌNH: khiên bảo vệ bên phải, hai dòng chữ xếp bên trái
function Shot4_4({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const first = interpolate(frame, [0, duration * 0.2], [0, 1], clamp);
  const second = interpolate(frame, [duration * 0.3, duration * 0.5], [0, 1], clamp);
  return (
    <AbsoluteFill>
      <Shield x={1380} y={540} size={440} color={PALETTE.accent} />
      <div style={{position: 'absolute', left: 200, top: 380, width: 720, fontSize: 52, fontWeight: 700, lineHeight: 1.25, color: PALETTE.ink, opacity: first}}>
        Đánh răng hai lần mỗi ngày
      </div>
      <div style={{position: 'absolute', left: 200, top: 560, width: 720, fontSize: 44, lineHeight: 1.25, color: PALETTE.ink, opacity: second}}>
        Kem có fluor giúp men chắc hơn
      </div>
    </AbsoluteFill>
  );
}

// Shot 4.5 — MÁY: toàn cảnh | HÌNH: căn phòng, bạn nhỏ cầm kẹo, bong bóng thoại
function Shot4_5({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const pop = spring({frame: frame - Math.round(duration * 0.3), fps, config: {damping: 12}});
  return (
    <AbsoluteFill>
      <Backdrop color="#FFE3A3" floor="#E8B96A" floorY={820} />
      <Person x={760} y={580} size={520} pose="stand" mood="happy" age="child" />
      <Candy x={960} y={640} size={120} />
      <Bubble x={1320} y={340} w={420} h={170} scale={pop} fontSize={40}>
        Ngọt quá!
      </Bubble>
    </AbsoluteFill>
  );
}

// Shot 4.6 — MÁY: đứng yên | HÌNH: màn hình chia đôi, trái kẹo, phải táo, nhãn dưới mỗi bên
function Shot4_6({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const show = interpolate(frame, [0, duration * 0.25], [0, 1], clamp);
  return (
    <AbsoluteFill>
      <Panel x={120} y={120} w={820} h={840} color={PALETTE.muted} radius={32} />
      <Panel x={980} y={120} w={820} h={840} color={PALETTE.muted} radius={32} />
      <Lollipop x={530} y={480} size={380} />
      <Apple x={1390} y={480} size={360} />
      <div style={{position: 'absolute', left: 230, top: 760, width: 600, textAlign: 'center', fontSize: 48, fontWeight: 700, color: PALETTE.ink, opacity: show}}>
        Kẹo dính răng
      </div>
      <div style={{position: 'absolute', left: 1090, top: 760, width: 600, textAlign: 'center', fontSize: 48, fontWeight: 700, color: PALETTE.ink, opacity: show}}>
        Táo làm sạch răng
      </div>
    </AbsoluteFill>
  );
}

// Shot 5.1 — MÁY: toàn cảnh, đứng yên | HÌNH: bạn nhỏ vẫy tay, tiêu đề hiện phía trên
function Shot5_1({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const grow = spring({frame, fps, config: {damping: 200}});
  const titleIn = interpolate(frame, [duration * 0.1, duration * 0.3], [0, 1], clamp);
  const {x, y, size} = LAYOUT.hero;
  return (
    <AbsoluteFill>
      <Person x={x} y={y + 60} size={size} pose="wave" mood="happy" age="child" scale={grow} />
      <div style={{position: 'absolute', left: 460, top: 130, width: 1000, textAlign: 'center', fontSize: 60, fontWeight: 700, lineHeight: 1.2, color: PALETTE.ink, opacity: titleIn}}>
        Vì sao răng bị sâu?
      </div>
    </AbsoluteFill>
  );
}

// Shot 5.2 — MÁY: đẩy vào chậm | HÌNH: chiếc răng bên trái, nhãn bên phải
function Shot5_2({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const zoom = interpolate(frame, [0, duration * 0.8], [1, 1.1], {...clamp, easing: Easing.inOut(Easing.cubic)});
  const labelIn = interpolate(frame, [duration * 0.2, duration * 0.4], [0, 1], clamp);
  const {x, y, size} = LAYOUT.tooth;
  return (
    <AbsoluteFill>
      <div style={{position: 'absolute', inset: 0, transformOrigin: `${x}px ${y}px`, transform: `scale(${zoom})`}}>
        <Tooth x={x} y={y} size={size} decay={0.2} />
        <div style={{position: 'absolute', left: LAYOUT.label.x - 180, top: LAYOUT.label.y - 30, width: LAYOUT.label.w, fontSize: 48, fontWeight: 700, lineHeight: 1.25, color: PALETTE.ink, opacity: labelIn}}>
          Men răng
        </div>
      </div>
    </AbsoluteFill>
  );
}

// Shot 5.3 — MÁY: trung cảnh | HÌNH: vi khuẩn trên răng, mũi tên chỉ vào, mèo suy nghĩ bên phải
function Shot5_3({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const arrow = interpolate(frame, [duration * 0.2, duration * 0.5], [0, 1], clamp);
  const {x, y, size} = LAYOUT.tooth;
  return (
    <AbsoluteFill>
      <Tooth x={x} y={y} size={size} decay={0.5} />
      <Germ x={x + 60} y={y - 120} size={140} />
      <svg width={300} height={80} viewBox="0 0 300 80" style={{position: 'absolute', left: 860, top: 400, opacity: arrow}}>
        <line x1={290} y1={40} x2={20} y2={40} stroke={PALETTE.accent} strokeWidth={10} strokeLinecap="round" />
        <polygon points="0,40 36,16 36,64" fill={PALETTE.accent} />
      </svg>
      <LottieClip id="cat.thinking" x={LAYOUT.cat.x} y={LAYOUT.cat.y} size={LAYOUT.cat.size} />
    </AbsoluteFill>
  );
}

// Shot 5.4 — MÁY: đứng yên | HÌNH: khiên bảo vệ bên phải, hai dòng chữ xếp bên trái
function Shot5_4({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const first = interpolate(frame, [0, duration * 0.2], [0, 1], clamp);
  const second = interpolate(frame, [duration * 0.3, duration * 0.5], [0, 1], clamp);
  return (
    <AbsoluteFill>
      <Shield x={1380} y={540} size={440} color={PALETTE.accent} />
      <div style={{position: 'absolute', left: 200, top: 380, width: 720, fontSize: 52, fontWeight: 700, lineHeight: 1.25, color: PALETTE.ink, opacity: first}}>
        Đánh răng hai lần mỗi ngày
      </div>
      <div style={{position: 'absolute', left: 200, top: 560, width: 720, fontSize: 44, lineHeight: 1.25, color: PALETTE.ink, opacity: second}}>
        Kem có fluor giúp men chắc hơn
      </div>
    </AbsoluteFill>
  );
}

// Shot 5.5 — MÁY: toàn cảnh | HÌNH: căn phòng, bạn nhỏ cầm kẹo, bong bóng thoại
function Shot5_5({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const pop = spring({frame: frame - Math.round(duration * 0.3), fps, config: {damping: 12}});
  return (
    <AbsoluteFill>
      <Backdrop color="#FFE3A3" floor="#E8B96A" floorY={820} />
      <Person x={760} y={580} size={520} pose="stand" mood="happy" age="child" />
      <Candy x={960} y={640} size={120} />
      <Bubble x={1320} y={340} w={420} h={170} scale={pop} fontSize={40}>
        Ngọt quá!
      </Bubble>
    </AbsoluteFill>
  );
}

// Shot 5.6 — MÁY: đứng yên | HÌNH: màn hình chia đôi, trái kẹo, phải táo, nhãn dưới mỗi bên
function Shot5_6({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const show = interpolate(frame, [0, duration * 0.25], [0, 1], clamp);
  return (
    <AbsoluteFill>
      <Panel x={120} y={120} w={820} h={840} color={PALETTE.muted} radius={32} />
      <Panel x={980} y={120} w={820} h={840} color={PALETTE.muted} radius={32} />
      <Lollipop x={530} y={480} size={380} />
      <Apple x={1390} y={480} size={360} />
      <div style={{position: 'absolute', left: 230, top: 760, width: 600, textAlign: 'center', fontSize: 48, fontWeight: 700, color: PALETTE.ink, opacity: show}}>
        Kẹo dính răng
      </div>
      <div style={{position: 'absolute', left: 1090, top: 760, width: 600, textAlign: 'center', fontSize: 48, fontWeight: 700, color: PALETTE.ink, opacity: show}}>
        Táo làm sạch răng
      </div>
    </AbsoluteFill>
  );
}

const SHOTS: React.FC<ShotProps>[] = [Shot1_1, Shot1_2, Shot1_3, Shot1_4, Shot1_5, Shot1_6, Shot2_1, Shot2_2, Shot2_3, Shot2_4, Shot2_5, Shot2_6, Shot3_1, Shot3_2, Shot3_3, Shot3_4, Shot3_5, Shot3_6, Shot4_1, Shot4_2, Shot4_3, Shot4_4, Shot4_5, Shot4_6, Shot5_1, Shot5_2, Shot5_3, Shot5_4, Shot5_5, Shot5_6];

function CreatorComposition({segments = []}: {segments?: {startFrame: number; durationInFrames: number}[]}) {
  return (
    <Stage>
      <Segments segments={segments}>
        {(index, segment) => {
          const Shot = SHOTS[index];
          return Shot ? <Shot duration={segment.durationInFrames} /> : null;
        }}
      </Segments>
    </Stage>
  );
}

registerRoot(() => (
  <Composition
    id="creator"
    component={CreatorComposition}
    width={1920}
    height={1080}
    fps={30}
    durationInFrames={150}
    calculateMetadata={calculateMetadataFromSegments}
  />
));
