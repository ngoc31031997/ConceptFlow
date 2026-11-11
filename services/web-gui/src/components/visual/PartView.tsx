import type { ReactNode } from "react";
import { asList, asObj, asText, type FieldSpec } from "./conceptSchema";
import researchStyles from "../research/research.module.css";
import styles from "./visual.module.css";

const HEX = /^#[0-9a-fA-F]{6}$/;

function Value({ spec, value }: { spec: FieldSpec; value: unknown }): ReactNode {
  switch (spec.kind) {
    case "lines": {
      const items = asList(value).map(asText).filter(Boolean);
      if (items.length === 0) return <span className={researchStyles.hint}>(trống)</span>;
      return (
        <ul className={styles.list}>
          {items.map((t, i) => (
            <li key={i}>{t}</li>
          ))}
        </ul>
      );
    }
    case "bool":
      return value ? "có" : "không";
    case "enum":
      return <span className={styles.tag}>{asText(value)}</span>;
    case "object":
      return (
        <dl className={styles.kv}>
          {spec.fields.map((f) => (
            <Row key={f.key} spec={f} value={asObj(value)[f.key]} />
          ))}
        </dl>
      );
    case "rows": {
      const rows = asList(value).map(asObj);
      if (rows.length === 0) return <span className={researchStyles.hint}>(trống)</span>;
      return (
        <div className={researchStyles.scroll}>
          <table className={researchStyles.table}>
            <thead>
              <tr>
                {spec.columns.map((c) => (
                  <th key={c.key}>{c.label}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, i) => (
                <tr key={i}>
                  {spec.columns.map((c) => (
                    <td key={c.key}>
                      <Value spec={c} value={row[c.key]} />
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      );
    }
    default: {
      const text = asText(value);
      if (HEX.test(text)) {
        return (
          <span>
            <i className={styles.swatch} style={{ background: text }} aria-hidden="true" />
            <span className={researchStyles.mono}>{text}</span>
          </span>
        );
      }
      return text || <span className={researchStyles.hint}>(trống)</span>;
    }
  }
}

function Row({ spec, value, mine }: { spec: FieldSpec; value: unknown; mine?: boolean }) {
  return (
    <>
      <dt>
        {spec.label}
        {mine && <span className={styles.mine}>· bạn sửa</span>}
      </dt>
      <dd>
        <Value spec={spec} value={value} />
      </dd>
    </>
  );
}

/**
 * One part of the visual concept, read only: every field of its schema, the
 * fields the Creator wrote marked as theirs.
 */
export function PartView({
  schema,
  content,
  mine,
}: {
  schema: readonly FieldSpec[];
  content: Record<string, unknown>;
  mine: string[];
}) {
  return (
    <dl className={styles.kv}>
      {schema.map((f) => (
        <Row key={f.key} spec={f} value={content[f.key]} mine={mine.includes(f.key)} />
      ))}
    </dl>
  );
}
