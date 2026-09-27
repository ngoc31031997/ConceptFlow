/**
 * A hand-written sample of what an illustrated script should look like
 * (CR-043): four shots of a "tooth decay" video, built only from the
 * conceptflow-mini illustration kit, in the same shape a generated script has
 * (PALETTE, LAYOUT, one ShotN_M per narration line, <Stage>/<Segments>).
 *
 * Render a preview with:  node render_gallery.mjs demo out.mp4
 */
import React from 'react';
import {registerRoot, Composition, AbsoluteFill, interpolate, spring, Easing, useCurrentFrame, useVideoConfig} from 'remotion';
import {calculateMetadataFromSegments, Segments} from '../conceptflow-mini/segments';
import {Stage} from '../conceptflow-mini/primitives';
import {
  Backdrop,
  Panel,
  Person,
  Table,
  Clock,
  Candy,
  Lollipop,
  Tooth,
  Germ,
  Drop,
  Bubble,
  Toothbrush,
  Shield,
  Sparkle,
} from '../conceptflow-mini/illustration';

const PALETTE = {
  room: '#FFC857', // căn phòng đời thường
  roomFloor: '#FFB347',
  stripe: '#FF7A45', // vạch ngăn hai thế giới
  inside: '#140B3A', // thế giới bên trong miệng
  villain: '#8BC34A', // vi khuẩn
  acid: '#C6F04A', // axit
  hero: '#3D7BFF', // bảo vệ
  ink: '#3A1F4B',
};

const LAYOUT = {
  kid: {x: 400, y: 610, size: 440},
  tooth: {x: 1380, y: 520, size: 360},
  bigTooth: {x: 960, y: 500, size: 520},
};

const clamp = {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'} as const;

export const narrations: string[] = [
  'Mỗi viên kẹo bạn ăn không chỉ nuôi bạn.',
  'Nó còn nuôi cả một đội quân vi khuẩn trong miệng.',
  'Chúng ăn đường, rồi thải ra axit, và axit gặm dần men răng.',
  'Tin tốt là, hai phút đánh răng đủ để quét sạch cả đội quân đó.',
];

type ShotProps = {duration: number};

// Shot 1.1 — MÁY: toàn cảnh, đứng yên | HÌNH: màn chia đôi — bên trái em bé ngồi bàn ăn kẹo, đồng hồ trên tường; bên phải thế giới trong miệng, một chiếc răng khoẻ đang cười
function Shot1_1({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const candyIn = spring({frame: frame - duration * 0.2, fps, config: {damping: 14}});
  const toothIn = spring({frame: frame - duration * 0.35, fps, config: {damping: 16}});
  const {kid, tooth} = LAYOUT;
  return (
    <AbsoluteFill>
      <Backdrop color={PALETTE.room} floor={PALETTE.roomFloor} floorY={830} />
      <Panel x={930} y={0} w={60} h={1080} color={PALETTE.stripe} />
      <Panel x={990} y={0} w={930} h={1080} color={PALETTE.inside} />
      <Clock x={340} y={200} size={190} hour={4} minute={interpolate(frame, [0, duration], [0, 20], clamp)} />
      <Table x={560} y={740} size={420} />
      <Person {...kid} pose="sit" age="child" mood="happy" shirt="#2ECC71" />
      <Candy x={600} y={600 - candyIn * 20} size={150} opacity={candyIn} rotate={-10} />
      <Lollipop x={730} y={590} size={170} opacity={candyIn} rotate={15} />
      <Tooth {...tooth} decay={0} shine scale={toothIn} />
    </AbsoluteFill>
  );
}

// Shot 1.2 — MÁY: đẩy chậm vào chiếc răng | HÌNH: vi khuẩn lần lượt bò vào từ mép phải, vây quanh răng; răng chuyển sang lo lắng
function Shot1_2({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const zoom = interpolate(frame, [0, duration], [1, 1.12], {...clamp, easing: Easing.inOut(Easing.cubic)});
  const {kid, tooth} = LAYOUT;
  const germs = [
    {dx: -250, dy: -180, v: 0, s: 130},
    {dx: 230, dy: -160, v: 1, s: 150},
    {dx: 260, dy: 140, v: 2, s: 120},
    {dx: -220, dy: 190, v: 3, s: 140},
  ];
  return (
    <AbsoluteFill>
      <Backdrop color={PALETTE.room} floor={PALETTE.roomFloor} floorY={830} />
      <Clock x={340} y={200} size={190} hour={4} minute={20} />
      <Table x={560} y={740} size={420} />
      <Person {...kid} pose="sit" age="child" mood="happy" shirt="#2ECC71" talking />
      <Panel x={930} y={0} w={60} h={1080} color={PALETTE.stripe} />
      <Panel x={990} y={0} w={930} h={1080} color={PALETTE.inside}>
        <div style={{position: 'absolute', inset: 0, transformOrigin: `${tooth.x - 990}px ${tooth.y}px`, transform: `scale(${zoom})`}}>
          <Tooth x={tooth.x - 990} y={tooth.y} size={tooth.size} decay={0} mood={frame > duration * 0.5 ? 'worried' : 'happy'} />
          {germs.map((g, i) => {
            const t = interpolate(frame, [duration * (0.1 + i * 0.12), duration * (0.4 + i * 0.12)], [0, 1], {
              ...clamp,
              easing: Easing.out(Easing.back(1.4)),
            });
            return (
              <Germ
                key={i}
                x={tooth.x - 990 + g.dx + (1 - t) * 700}
                y={tooth.y + g.dy}
                size={g.s}
                variant={g.v}
                color={PALETTE.villain}
                opacity={t}
              />
            );
          })}
        </div>
      </Panel>
    </AbsoluteFill>
  );
}

// Shot 2.1 — MÁY: cận cảnh răng lớn giữa khung (đi xuyên qua nửa phải) | HÌNH: vi khuẩn nhả giọt axit rơi xuống mặt răng, vết sâu lan dần, răng đau
function Shot2_1({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const decay = interpolate(frame, [duration * 0.25, duration * 0.85], [0, 0.9], clamp);
  const {bigTooth} = LAYOUT;
  const drops = [0, 1, 2];
  return (
    <AbsoluteFill>
      <Backdrop color={PALETTE.inside} />
      <Tooth {...bigTooth} decay={decay} />
      <Germ x={660} y={230} size={170} color={PALETTE.villain} variant={1} />
      <Germ x={1270} y={210} size={150} color={PALETTE.villain} variant={0} flip />
      {drops.map((i) => {
        const t = ((frame - i * 12) % 36) / 36;
        const visible = frame > duration * 0.15 && frame < duration * 0.9;
        return (
          <Drop
            key={i}
            x={(i % 2 ? 1230 : 700) + i * 10}
            y={interpolate(t, [0, 1], [300, 420])}
            size={60}
            color={PALETTE.acid}
            opacity={visible ? 1 - t * 0.6 : 0}
          />
        );
      })}
      <Bubble x={1500} y={620} w={300} h={120} opacity={interpolate(frame, [duration * 0.6, duration * 0.7], [0, 1], clamp)} textColor={PALETTE.ink}>
        Axit!
      </Bubble>
    </AbsoluteFill>
  );
}

// Shot 3.1 — MÁY: trung cảnh | HÌNH: bàn chải trượt vào từ phải, chà qua lại; vi khuẩn bị hất văng ra khỏi khung; răng sáng lại, khiên bảo vệ hiện lên
function Shot3_1({duration}: ShotProps) {
  const frame = useCurrentFrame();
  const {fps} = useVideoConfig();
  const {bigTooth} = LAYOUT;
  const enter = interpolate(frame, [0, duration * 0.2], [900, 0], {...clamp, easing: Easing.out(Easing.cubic)});
  const scrub = frame > duration * 0.2 && frame < duration * 0.6 ? Math.sin(frame / 2) * 60 : 0;
  const clean = interpolate(frame, [duration * 0.3, duration * 0.6], [0.9, 0], clamp);
  const fly = interpolate(frame, [duration * 0.3, duration * 0.55], [0, 1], {...clamp, easing: Easing.in(Easing.quad)});
  const shield = spring({frame: frame - duration * 0.65, fps, config: {damping: 12}});
  return (
    <AbsoluteFill>
      <Backdrop color={PALETTE.inside} />
      <Tooth {...bigTooth} decay={clean} shine={clean < 0.1} mood={clean < 0.1 ? 'happy' : 'surprised'} />
      <Germ x={660 - fly * 700} y={230 - fly * 300} size={170} color={PALETTE.villain} variant={1} rotate={fly * -200} mood="sad" />
      <Germ x={1270 + fly * 700} y={210 - fly * 260} size={150} color={PALETTE.villain} rotate={fly * 220} mood="sad" />
      <Toothbrush x={1060 + enter + scrub} y={250} size={520} color={PALETTE.hero} rotate={-6} bristlesDown />
      <Shield x={1420} y={640} size={200} color={PALETTE.hero} scale={shield} />
      {clean < 0.1 ? (
        <>
          <Sparkle x={700} y={480} size={70} />
          <Sparkle x={1200} y={700} size={50} />
        </>
      ) : null}
    </AbsoluteFill>
  );
}

const SHOTS: React.FC<ShotProps>[] = [Shot1_1, Shot1_2, Shot2_1, Shot3_1];

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

const DEMO_SEGMENTS = [0, 1, 2, 3].map((i) => ({startFrame: i * 120, durationInFrames: 120}));

registerRoot(() => (
  <Composition
    id="creator"
    component={CreatorComposition}
    width={1920}
    height={1080}
    fps={30}
    durationInFrames={480}
    defaultProps={{segments: DEMO_SEGMENTS}}
    calculateMetadata={calculateMetadataFromSegments}
  />
));
