#!/usr/bin/env node
/**
 * Node driver for the real render pass — feature/remotion-engine.
 *
 * `remotion_renderer.py`'s dry_run() reads `narrations` straight off the
 * script's source text (no Node process at all — see that file's docstring),
 * so this script only ever runs for a REAL render: bundle -> selectComposition
 * -> renderMedia, exactly the three-step flow documented at
 * https://www.remotion.dev/docs/ssr-node.
 *
 * Args: --entry <path to the Creator's .tsx, already written to disk>
 *       --id <composition id, from script-processing's parsed scene_class_name>
 *       --props <path to a JSON file: {"segments": [{startFrame, durationInFrames}, ...]}>
 *       --out <path to write the .mp4>
 */
import {bundle} from '@remotion/bundler';
import {renderMedia, selectComposition} from '@remotion/renderer';
import {readFileSync} from 'node:fs';

function parseArgs(argv) {
  const out = {};
  for (let i = 0; i < argv.length; i += 2) {
    out[argv[i].replace(/^--/, '')] = argv[i + 1];
  }
  return out;
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  const inputProps = JSON.parse(readFileSync(args.props, 'utf-8'));

  const serveUrl = await bundle({entryPoint: args.entry});

  const composition = await selectComposition({
    serveUrl,
    id: args.id,
    inputProps,
  });

  await renderMedia({
    composition,
    serveUrl,
    codec: 'h264',
    outputLocation: args.out,
    inputProps,
    // Recommended by Remotion's own Docker guide
    // (https://www.remotion.dev/docs/docker) for containers without a full
    // desktop compositor.
    chromiumOptions: {enableMultiProcessOnLinux: true},
  });
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
