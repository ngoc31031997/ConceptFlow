/**
 * Contact sheets of the flat illustration kit (CR-043), for reviewing the
 * drawings by eye: `node render_gallery.mjs sheets <outDir>`.
 */
import React from 'react';
import {registerRoot, Composition, AbsoluteFill} from 'remotion';
import {Stage} from '../conceptflow-mini/primitives';
import * as K from '../conceptflow-mini/illustration';

function Label({x, y, children}: {x: number; y: number; children: React.ReactNode}) {
  return <div style={{position: 'absolute', left: x - 150, top: y, width: 300, textAlign: 'center', fontSize: 26, color: '#3A1F4B', fontWeight: 700}}>{children}</div>;
}

function People() {
  const poses: K.PersonPose[] = ['stand', 'sit', 'wave', 'point', 'think', 'cheer', 'ouch', 'shrug', 'walk'];
  return (
    <Stage>
      <K.Backdrop color="#FFC857" floor="#FFB347" floorY={900} />
      {poses.map((p, i) => (
        <React.Fragment key={p}>
          <K.Person x={130 + i * 205} y={420} size={380} pose={p} mood={(['neutral','happy','sad','worried','surprised','pain','angry','happy','neutral'] as K.Mood[])[i]} age={i === 1 ? 'elder' : i === 4 ? 'child' : 'adult'} outfit={i === 3 ? 'coat' : i === 5 ? 'dress' : 'shirt'} hairStyle={i === 5 ? 'long' : i === 7 ? 'bun' : undefined} shirt={i === 1 ? '#2ECC71' : undefined} />
          <Label x={130 + i * 205} y={620}>{p}</Label>
        </React.Fragment>
      ))}
      {(['neutral','happy','sad','worried','surprised','pain','angry'] as K.Mood[]).map((m, i) => (
        <React.Fragment key={m}>
          <K.Person x={180 + i * 260} y={860} size={300} mood={m} talking={false} still />
        </React.Fragment>
      ))}
    </Stage>
  );
}

function Things() {
  const items: [string, React.ReactNode][] = [
    ['Tooth 0', <K.Tooth size={200} decay={0} shine />],
    ['Tooth .3', <K.Tooth size={200} decay={0.3} />],
    ['Tooth .6', <K.Tooth size={200} decay={0.6} />],
    ['Tooth 1', <K.Tooth size={200} decay={1} />],
    ['Germ', <K.Germ size={180} />],
    ['Germ 1', <K.Germ size={180} variant={1} color="#B06CE0" />],
    ['Germ happy', <K.Germ size={180} variant={2} color="#4FC3F7" mood="happy" />],
    ['OpenMouth', <K.OpenMouth size={280} decayed={[1, 8]} />],
    ['Toothbrush', <K.Toothbrush size={260} />],
    ['Toothpaste', <K.Toothpaste size={240} />],
    ['Drop', <K.Drop size={140} face />],
    ['Shield', <K.Shield size={170} />],
    ['Candy', <K.Candy size={200} />],
    ['Lollipop', <K.Lollipop size={200} />],
    ['Soda', <K.Soda size={200} />],
    ['Donut', <K.Donut size={180} />],
    ['Apple', <K.Apple size={170} />],
    ['Clock', <K.Clock size={180} hour={4} />],
    ['Table', <K.Table size={260} />],
    ['Chair', <K.Chair size={200} />],
    ['Window', <K.Window size={200} time="night" />],
    ['Plant', <K.Plant size={200} />],
    ['House', <K.House size={200} />],
    ['Tree', <K.Tree size={200} />],
    ['Sun', <K.Sun size={170} />],
    ['Cloud', <K.Cloud size={200} />],
    ['Lightbulb', <K.Lightbulb size={190} />],
    ['Heart', <K.Heart size={160} />],
    ['Coin', <K.Coin size={140} />],
    ['Book', <K.Book size={170} />],
    ['Phone', <K.Phone size={180} />],
    ['Magnifier', <K.Magnifier size={170} />],
    ['Mark', <K.Mark size={120} />],
    ['Sparkle', <K.Sparkle size={100} />],
    ['Airplane', <K.Airplane size={260} />],
  ];
  const cols = 8;
  return (
    <Stage>
      <K.Backdrop color="#FFE3A3" />
      {items.map(([name, el], i) => {
        const cx = 120 + (i % cols) * 240;
        const cy = 100 + Math.floor(i / cols) * 205;
        return (
          <React.Fragment key={name}>
            {React.cloneElement(el as React.ReactElement, {x: cx, y: cy})}
            <Label x={cx} y={cy + 78}>{name}</Label>
          </React.Fragment>
        );
      })}
      <K.Bubble x={1700} y={960} w={300} h={100} flip>Đường!</K.Bubble>
    </Stage>
  );
}

registerRoot(() => (
  <>
    <Composition id="people" component={People} width={1920} height={1080} fps={30} durationInFrames={120} />
    <Composition id="things" component={Things} width={1920} height={1080} fps={30} durationInFrames={120} />
  </>
));
