# Sequence Flows — Unit 3: TTS Service

## Flow 1: Synthesize Speech (New Audio)

```mermaid
sequenceDiagram
    participant RD as Rendering Service
    participant API as adapters/api/router.py
    participant UC as SynthesizeSpeechUseCase
    participant AP as artifact_paths.py
    participant ENG as PiperTTSAdapter
    participant FS as Shared Volume

    RD->>API: POST /v1/tts/synthesize (X-Saga-ID header, project_id, scene_index, text, language)
    API->>API: validate language in {vi, en}
    alt invalid language
        API-->>RD: 400 unsupported_language
    end
    API->>UC: synthesize(SpeechRequest)
    UC->>AP: compute_path(project_id, scene_index, language)
    AP-->>UC: /shared/{project_id}/audio/{scene_index}_{language}.wav
    UC->>FS: check file exists
    alt file already exists (idempotent hit)
        FS-->>UC: exists
        UC->>FS: read duration from existing file
        UC-->>API: SpeechResult(audio_path, duration_seconds)
    else file does not exist
        UC->>ENG: synthesize(text, language, output_path)
        ENG->>FS: write .wav file
        alt engine failure
            ENG-->>UC: raise TTSEngineError
            UC-->>API: propagate error
            API-->>RD: 502 tts_engine_failure
        else success
            ENG-->>UC: duration_seconds
            UC-->>API: SpeechResult(audio_path, duration_seconds)
        end
    end
    API-->>RD: 200 {audio_path, duration_seconds}
```

## Flow 2: Idempotent Retry (Rendering Service retries after transient failure elsewhere in the scene pipeline)

```mermaid
sequenceDiagram
    participant RD as Rendering Service
    participant API as adapters/api/router.py
    participant UC as SynthesizeSpeechUseCase
    participant FS as Shared Volume

    RD->>API: POST /v1/tts/synthesize (same project_id + scene_index as before)
    API->>UC: synthesize(SpeechRequest)
    UC->>FS: check file exists at conventional path
    FS-->>UC: exists (from prior successful call)
    UC-->>API: SpeechResult(audio_path, duration_seconds) — read directly, no re-synthesis
    API-->>RD: 200 {audio_path, duration_seconds}
```

**Note**: Flow 2 covers the case where `render_scenes` step fails downstream (e.g., another scene fails) and Orchestrator/Rendering Service retries only the failing scene, per the compensating-action strategy in `services.md`. TTS Service never re-synthesizes audio it has already produced for a given `project_id`+`scene_index`.
