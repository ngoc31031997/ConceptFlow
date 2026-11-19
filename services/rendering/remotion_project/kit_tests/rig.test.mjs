// Rig maths: IK, poses, easing, cycles, mechanism parts.
import assert from 'node:assert/strict';
import {test} from 'node:test';
import {load, near} from './load_kit.mjs';

// An arm hanging straight down.
const joints = {root: [0, 0], mid: [0, 100], tip: [0, 200]};

/** Forward kinematics of the two bones, to check the IK against. */
function tipAfter(j, ik) {
  const rad = (d) => (d * Math.PI) / 180;
  const rot = (v, d) => [v[0] * Math.cos(rad(d)) - v[1] * Math.sin(rad(d)), v[0] * Math.sin(rad(d)) + v[1] * Math.cos(rad(d))];
  const u = rot([j.mid[0] - j.root[0], j.mid[1] - j.root[1]], ik.upper);
  const f = rot([j.tip[0] - j.mid[0], j.tip[1] - j.mid[1]], ik.upper + ik.lower);
  return [j.root[0] + u[0] + f[0], j.root[1] + u[1] + f[1]];
}

test('two-bone IK lands the hand on a reachable target, for both bend directions', async () => {
  const {ikTwoBone} = await load('skeleton.tsx');
  for (const bend of [1, -1]) {
    for (const target of [[120, 80], [-90, 60], [60, -50], [0, 150]]) {
      const tip = tipAfter(joints, ikTwoBone(joints, target, bend));
      near(tip[0], target[0], 1e-6);
      near(tip[1], target[1], 1e-6);
    }
  }
});

test('two-bone IK stretches toward an unreachable target without breaking', async () => {
  const {ikTwoBone} = await load('skeleton.tsx');
  const tip = tipAfter(joints, ikTwoBone(joints, [1000, 0]));
  assert.ok(tip[0] > 199 && Math.abs(tip[1]) < 1, `tip ${tip}`);
});

test('two-bone IK rejects a limb with a zero-length bone', async () => {
  const {ikTwoBone} = await load('skeleton.tsx');
  assert.throws(() => ikTwoBone({root: [0, 0], mid: [0, 0], tip: [0, 10]}, [5, 5]), /độ dài 0/);
});

test('action easing starts and ends on its keyframes, steps back first and overshoots after', async () => {
  const {easeAction} = await load('skeleton.tsx');
  assert.equal(easeAction(0), 0);
  assert.equal(easeAction(1), 1);
  assert.ok(easeAction(0.07) < 0, 'wind-up goes the other way first');
  assert.ok(Math.max(...Array.from({length: 100}, (_, i) => easeAction((i + 1) / 100))) > 1, 'overshoots the target');
});

test('poses follow the actions in order and hold', async () => {
  const {poseAt} = await load('skeleton.tsx');
  const actions = [{at: 0.2, pose: 'point'}, {at: 0.6, pose: 'cheer', hold: 0.2}];
  near(poseAt('humanoid', 'stand', actions, 0).angles.armR, 0);
  near(poseAt('humanoid', 'stand', actions, 0.2).angles.armR, -85);
  near(poseAt('humanoid', 'stand', actions, 0.7).angles.armR, -160);
  near(poseAt('humanoid', 'stand', actions, 1).angles.armR, -160);
});

test('an unknown pose names the poses that exist', async () => {
  const {poseAt} = await load('skeleton.tsx');
  assert.throws(() => poseAt('humanoid', 'stand', [{at: 0.5, pose: 'moonwalk'}], 0.8), /moonwalk.*có:/);
});

test('walking swings the legs in opposite directions and repeats after a period', async () => {
  const {cycleAngles} = await load('skeleton.tsx');
  const a = cycleAngles('humanoid', {name: 'walk'}, 0.1);
  near(a.thighL, -a.thighR);
  near(a.thighL, cycleAngles('humanoid', {name: 'walk'}, 1.1).thighL);
  assert.deepEqual(cycleAngles('fish', {name: 'walk'}, 0.3), {});
});

test('a fish swims and a bird flaps', async () => {
  const {cycleAngles} = await load('skeleton.tsx');
  assert.ok(Math.abs(cycleAngles('fish', {name: 'swim'}, 0.25).tail) > 1);
  assert.ok(Math.abs(cycleAngles('bird', {name: 'fly'}, 0.05).wingL) > 1);
});

test('child bones trail their parents', async () => {
  const {solveRig, poseAt} = await load('skeleton.tsx');
  const motion = {skeleton: 'humanoid', actions: [{at: 0.5, pose: 'cheer'}], duration: 100, still: true};
  const f = solveRig(motion, 45, 30);
  const target = poseAt('humanoid', 'stand', motion.actions, 0.45).angles;
  // the arm (depth 2) is two lags behind the current pose; the spine (depth 1) has none of the arm's pose
  assert.ok(Math.abs(f.angles.armL - target.armL) > 1e-6, 'arm lags behind the pose of this frame');
});

test('actions without a shot duration are an error', async () => {
  const {solveRig} = await load('skeleton.tsx');
  assert.throws(() => solveRig({skeleton: 'humanoid', actions: [{at: 0.5, pose: 'wave'}]}, 3, 30), /duration/);
});

test('every skeleton has one root, a rest pose, and poses only move its own bones', async () => {
  const {SKELETONS, POSES, restPoseName} = await load('skeleton.tsx');
  for (const [name, tree] of Object.entries(SKELETONS)) {
    assert.ok(POSES[name][restPoseName(name)], `${name} has a rest pose`);
    assert.equal(Object.values(tree).filter((p) => p === null).length, 1, `${name} has one root`);
    for (const [pose, def] of Object.entries(POSES[name])) {
      for (const bone of Object.keys(def.angles)) assert.ok(bone in tree, `${name}/${pose} moves unknown bone ${bone}`);
    }
  }
});

test('parts turn, swing and slide as declared', async () => {
  const {partTransform} = await load('skeleton.tsx');
  assert.equal(partTransform('rotate', [10, 20], 0.5, {speed: 720}), 'rotate(360 10 20)');
  assert.equal(partTransform('none', [0, 0], 1, {}), '');
  assert.match(partTransform('slide', [0, 0], 1, {offset: [30, 0]}), /^translate\(/);
});

test('the head turns toward what it looks at, within a limit', async () => {
  const {lookAngle} = await load('skeleton.tsx');
  assert.ok(lookAngle([0, 0], [500, 0]) > 0);
  assert.ok(lookAngle([0, 0], [-500, 0]) < 0);
  assert.ok(lookAngle([0, 0], [99999, 0]) <= 30);
});

test('the elbow keeps both bone lengths and bends to the chosen side', async () => {
  const {elbowPosition} = await load('skeleton.tsx');
  const root = [0, 0];
  const target = [120, 0];
  const up = elbowPosition(root, target, 100, 100, 1);
  const down = elbowPosition(root, target, 100, 100, -1);
  near(Math.hypot(up[0], up[1]), 100, 1e-6);
  near(Math.hypot(up[0] - 120, up[1]), 100, 1e-6);
  assert.ok(up[1] * down[1] < 0, 'the two bends are on opposite sides');
});

const GOOD_KIT = {
  outline: 0, radius: 0.18, rimWidth: 0.035, rimStrength: 0.7, shadowStrength: 0.35, aoStrength: 0.25,
  specular: {glossy: 0.5, metal: 0.8, glass: 0.9},
  texture: {fur: 0.25, fabric: 0.12, wood: 0.2, stone: 0.22, organic: 0.15, leaf: 0.12},
  sss: 0.4, glowBlur: 0.08, bloom: 0.5,
  haze: {far: 0.35, mid: 0.15},
  dof: {sky: 6, far: 3, mid: 1, near: 10},
  grain: 0.05, vignette: 0.3, ink: '#141A3A',
  previewLook: {
    light: {x: 0.7, y: 0.2, color: '#FFF4D6', intensity: 0.8},
    ambient: '#8A93B8',
    atmosphere: {color: '#B8C0DC', density: 0.2},
    focus: 'figures',
    grade: {warmth: 0, contrast: 0, saturation: 0},
  },
};

test('a complete kit is accepted as given', async () => {
  const {parseKit} = await load('kits.ts');
  assert.equal(parseKit(GOOD_KIT), GOOD_KIT);
});

test('a missing kit or a bad field is an error that names the field', async () => {
  const {parseKit} = await load('kits.ts');
  assert.throws(() => parseKit(undefined), /Thiếu kit/);
  const without = structuredClone(GOOD_KIT);
  delete without.haze.mid;
  assert.throws(() => parseKit(without), /thiếu trường haze\.mid/);
  assert.throws(() => parseKit({...GOOD_KIT, grain: 1.5}), /kit\.grain phải trong khoảng 0\.\.1/);
  assert.throws(() => parseKit({...GOOD_KIT, dof: {...GOOD_KIT.dof, near: -1}}), /kit\.dof\.near/);
  assert.throws(() => parseKit({...GOOD_KIT, ink: 'red'}), /kit\.ink phải là màu #RRGGBB/);
  assert.throws(() => parseKit({...GOOD_KIT, previewLook: {...GOOD_KIT.previewLook, focus: 'back'}}), /kit\.previewLook\.focus/);
  assert.throws(() => parseKit({...GOOD_KIT, extra: 1}), /trường lạ: extra/);
  const grade = (g) => ({...GOOD_KIT, previewLook: {...GOOD_KIT.previewLook, grade: {...GOOD_KIT.previewLook.grade, ...g}}});
  assert.ok(parseKit(grade({warmth: -0.5})));
  assert.throws(() => parseKit(grade({warmth: 1.5})), /kit\.previewLook\.grade\.warmth phải trong khoảng -1\.\.1/);
});
