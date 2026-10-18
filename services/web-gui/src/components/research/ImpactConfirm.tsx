import type { ReactNode } from "react";
import { Button, Dialog, type ButtonVariant } from "../ui";
import styles from "./research.module.css";

/** One way out of a confirmation besides cancelling. */
export interface ConfirmChoice {
  label: string;
  variant: ButtonVariant;
  onChoose: () => void;
  testId: string;
}

interface ImpactConfirmProps {
  open: boolean;
  onClose: () => void;
  /** Names the item and the task ("Bỏ dữ kiện F08?"). */
  title: string;
  /** What is changed or deleted, by name. */
  lead?: ReactNode;
  /** What this does to the workflow, one line each (docs/ux-ui-design-rules.md, rule 5). */
  list: string[];
  /** Whether it can be undone, and how. */
  undo?: string;
  /** The ways to go on, the main one last. */
  choices: ConfirmChoice[];
  testId: string;
}

/**
 * The confirmation that rule 5 asks for before a change that deletes data,
 * sends later work back, or throws away what the Creator wrote: what changes,
 * what follows from it, whether it can be undone, then "Huỷ" and an action
 * named by what it does. Clicking outside closes it, since nothing is typed.
 */
export function ImpactConfirm({ open, onClose, title, lead, list, undo, choices, testId }: ImpactConfirmProps) {
  return (
    <Dialog
      open={open}
      onClose={onClose}
      role="alertdialog"
      size="sm"
      title={title}
      testId={testId}
      footer={
        <>
          <Button variant="ghost" onClick={onClose} data-testid={`${testId}-cancel`}>
            Huỷ
          </Button>
          {choices.map((choice) => (
            <Button
              key={choice.testId}
              variant={choice.variant}
              onClick={() => {
                choice.onChoose();
                onClose();
              }}
              data-testid={choice.testId}
            >
              {choice.label}
            </Button>
          ))}
        </>
      }
    >
      {lead && <p className={styles.lead}>{lead}</p>}
      {list.length > 0 && (
        <ul className={styles.confirmList} data-testid={`${testId}-list`}>
          {list.map((item) => (
            <li key={item}>{item}</li>
          ))}
        </ul>
      )}
      {undo && <p className={styles.undo}>{undo}</p>}
    </Dialog>
  );
}
