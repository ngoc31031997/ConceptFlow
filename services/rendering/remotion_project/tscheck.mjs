// Long-lived TypeScript checker for Creator/LLM-authored Remotion scripts
// (CR-039 FR104, kept warm so a check does not pay tsc's start-up every time).
//
// Protocol: one JSON object per line on stdin, one per line on stdout.
//   in:  {"id": "...", "code": "<tsx>"}
//   out: {"id": "...", "diagnostics": [{"line", "col", "code", "message"}], "other": ["..."]}
//    or: {"id": "...", "error": "..."}
// `diagnostics` are errors inside the script; `other` are errors tsc would
// also fail on that belong to no line of it (config, global) — the caller
// must never read those as a pass.
//
// The script is checked as if it were src/__tscheck__.tsx under the project's
// own tsconfig, so relative imports resolve exactly as at render time. It is
// never written to disk. Every other file (lib .d.ts, @types, remotion,
// conceptflow-mini) is parsed once and its SourceFile reused by later
// programs — the same sharing a language service does — which is where the
// start-up time of a plain `tsc` goes.

import path from 'node:path';
import readline from 'node:readline';
import ts from 'typescript';

const projectDir = path.resolve(process.argv[2] || process.cwd());
const configPath = path.join(projectDir, 'tsconfig.json');
const checkFile = path.join(projectDir, 'src', '__tscheck__.tsx');

function loadOptions() {
  const read = ts.readConfigFile(configPath, ts.sys.readFile);
  if (read.error) {
    throw new Error(ts.flattenDiagnosticMessageText(read.error.messageText, '\n'));
  }
  const parsed = ts.parseJsonConfigFileContent(read.config, ts.sys, projectDir, undefined, configPath);
  const errors = parsed.errors.filter((d) => d.category === ts.DiagnosticCategory.Error);
  if (errors.length) {
    throw new Error(errors.map((d) => ts.flattenDiagnosticMessageText(d.messageText, '\n')).join('; '));
  }
  return {...parsed.options, noEmit: true};
}

const options = loadOptions();
const base = ts.createCompilerHost(options, true);
const shared = new Map();
let current = '';
let previous;

const host = {
  ...base,
  fileExists: (f) => f === checkFile || base.fileExists(f),
  readFile: (f) => (f === checkFile ? current : base.readFile(f)),
  getSourceFile(fileName, languageVersion, onError, shouldCreate) {
    if (fileName === checkFile) {
      return ts.createSourceFile(fileName, current, languageVersion, true, ts.ScriptKind.TSX);
    }
    let sf = shared.get(fileName);
    if (!sf) {
      sf = base.getSourceFile(fileName, languageVersion, onError, shouldCreate);
      if (sf) shared.set(fileName, sf);
    }
    return sf;
  },
};

function message(d) {
  return ts.flattenDiagnosticMessageText(d.messageText, '\n');
}

function check(code) {
  current = code;
  const program = ts.createProgram({rootNames: [checkFile], options, host, oldProgram: previous});
  previous = program;
  const sf = program.getSourceFile(checkFile);
  const all = [
    ...program.getOptionsDiagnostics(),
    ...program.getGlobalDiagnostics(),
    ...program.getSyntacticDiagnostics(sf),
    ...program.getSemanticDiagnostics(sf),
  ].filter((d) => d.category === ts.DiagnosticCategory.Error);
  const diagnostics = [];
  const other = [];
  for (const d of all) {
    if (d.file && d.file.fileName === checkFile && d.start !== undefined) {
      const {line, character} = d.file.getLineAndCharacterOfPosition(d.start);
      diagnostics.push({line: line + 1, col: character + 1, code: `TS${d.code}`, message: message(d)});
    } else {
      other.push(`${d.file ? d.file.fileName + ': ' : ''}TS${d.code}: ${message(d)}`);
    }
  }
  return {diagnostics, other};
}

const rl = readline.createInterface({input: process.stdin, crlfDelay: Infinity});
rl.on('line', (line) => {
  if (!line.trim()) return;
  let id = null;
  let out;
  try {
    const req = JSON.parse(line);
    id = req.id ?? null;
    out = {id, ...check(String(req.code ?? ''))};
  } catch (err) {
    out = {id, error: String(err && err.stack ? err.stack : err)};
  }
  process.stdout.write(JSON.stringify(out) + '\n');
});
rl.on('close', () => process.exit(0));
