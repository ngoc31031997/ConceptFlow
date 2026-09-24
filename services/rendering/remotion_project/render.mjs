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
import {transform} from 'esbuild';

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

  // Fast syntax check before the (slow) bundle: a broken script fails here with
  // a precise, LLM-readable message instead of a webpack stack trace.
  try {
    await transform(readFileSync(args.entry, 'utf-8'), {
      loader: 'tsx',
      sourcefile: 'CreatorEntry.tsx',
    });
  } catch (err) {
    const msgs = (err.errors ?? []).map(
      (e) => `CreatorEntry.tsx:${e.location?.line}:${e.location?.column}: ${e.text}` +
        (e.location?.lineText ? `\n    ${e.location.lineText}` : ''),
    );
    console.error(`Script syntax error (esbuild):\n${msgs.join('\n') || err.message}`);
    process.exit(1);
  }

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
