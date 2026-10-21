import { useCallback, useEffect, useRef, useState } from "react";
import { scriptAudioUrl, type ScriptLine, type ScriptState } from "../api/client";

/** What the player does, for the buttons that start and stop it. */
export interface LinePlayer {
  /** The line whose voice is playing now. */
  playingId: string | null;
  /** The line the player waits on: its voice is being read. */
  waitingId: string | null;
  /** Why playing stopped, in words for the Creator. */
  error: string | null;
  /** Reads the lines that have no current voice file, then plays them one after another; replaces what was queued. */
  play: (lineIds: string[]) => Promise<void>;
  stop: () => void;
}

function lineOf(state: ScriptState | null, id: string): ScriptLine | undefined {
  for (const beat of state?.beats ?? []) {
    const line = beat.lines.find((l) => l.id === id);
    if (line) return line;
  }
  return undefined;
}

/**
 * Plays script lines in the browser. A line with a current voice file plays at
 * once; one without is sent to be read (`speak`) and plays when its file is
 * ready, which the screen learns by reloading `state`. A line that cannot be
 * read stops the queue with the reason, never a silent skip.
 */
export function useLinePlayer(projectId: string, state: ScriptState | null, speak: (lineIds: string[]) => Promise<boolean>): LinePlayer {
  const [queue, setQueue] = useState<string[]>([]);
  const [playingId, setPlayingId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  // Lines `speak` was called for, and those whose call has returned.
  const asked = useRef(new Set<string>());
  const answered = useRef(new Set<string>());
  const audio = useRef<HTMLAudioElement | null>(null);

  const stop = useCallback(() => {
    audio.current?.pause();
    audio.current = null;
    setQueue([]);
    setPlayingId(null);
  }, []);

  useEffect(() => () => audio.current?.pause(), []);

  const fail = useCallback(
    (message: string) => {
      stop();
      setError(message);
    },
    [stop],
  );

  useEffect(() => {
    if (queue.length === 0 || playingId !== null || !state) return;
    const id = queue[0];
    const line = lineOf(state, id);
    if (!line) {
      setQueue((q) => q.slice(1));
      return;
    }
    const { audio: reading } = line;
    if (reading.state === "ready" && reading.file) {
      const element = new Audio(scriptAudioUrl(projectId, reading.file));
      audio.current = element;
      setPlayingId(id);
      element.onended = () => {
        audio.current = null;
        setPlayingId(null);
        setQueue((q) => q.slice(1));
      };
      element.onerror = () => fail(`Không phát được giọng của câu ${id}.`);
      element.play().catch(() => fail(`Trình duyệt không cho phát giọng của câu ${id}.`));
      return;
    }
    if (reading.state === "failed") {
      fail(reading.error || `Không đọc được câu ${id}.`);
      return;
    }
    // Still waiting for the voice: "none" before the request was answered, pending or running after.
    if (reading.state === "none" && answered.current.has(id)) fail(`Câu ${id} chưa được gửi đi đọc.`);
  }, [queue, playingId, state, projectId, fail]);

  const play = useCallback(
    async (lineIds: string[]) => {
      stop();
      setError(null);
      const need = lineIds.filter((id) => {
        const state_ = lineOf(state, id)?.audio.state;
        return state_ === "none" || state_ === "failed";
      });
      need.forEach((id) => {
        asked.current.add(id);
        answered.current.delete(id);
      });
      setQueue(lineIds);
      if (need.length === 0) return;
      const ok = await speak(need);
      need.forEach((id) => answered.current.add(id));
      if (!ok) fail("Không gửi được câu đi đọc.");
    },
    [state, speak, stop, fail],
  );

  const head = queue[0] ?? null;
  return { playingId, waitingId: playingId === null ? head : null, error, play, stop };
}
