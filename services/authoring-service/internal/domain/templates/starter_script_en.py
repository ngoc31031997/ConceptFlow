"""
A short explainer on Java's `for` loop, written with the ConceptFlow design system.
"""

from conceptflow import *


class ForLoopIntroScene(ConceptFlowScene):
    def construct(self):
        # --- Opening ---------------------------------------------------------
        card = TitleCard("The for loop in Java", "Init · Condition · Step")
        self.reveal(card)
        self.narrate(
            "A for loop in Java has three parts: initialising a counter, a loop condition, and a step."
        )
        self.dismiss(card)

        # --- Concrete example first, definition after ------------------------
        panel = CodePanel(
            "for (int i = 0; i < 5; i++) {\n"
            "    System.out.println(i);\n"
            "}",
            "java",
        )
        self.reveal(panel)
        self.narrate("Look at this snippet first. It prints the numbers zero through four.")

        note = Callout("i runs 0 to 4, never reaching 5", tone="warning")
        note.next_to(panel, DOWN, buff=0.5)
        self.reveal(note)
        self.narrate(
            "The condition is i less than five, so the loop stops the moment i becomes five. "
            "Five is never printed."
        )
        self.dismiss(panel, note)

        # --- Draw out the pattern --------------------------------------------
        steps = StepList([
            "Init: runs exactly once, before anything else",
            "Condition: checked before every pass",
            "Body: runs while the condition holds",
            "Step: runs after every pass",
        ])
        self.reveal(steps)
        self.narrate(
            "The general pattern is four steps: initialise once, then check the condition, "
            "run the body, and advance the counter."
        )
        self.emphasize(steps)
        self.narrate("The order matters: the condition is always checked before the body runs.")
        self.dismiss(steps)

        # --- Compare ----------------------------------------------------------
        compare = ComparisonSplit(
            "for", "You know the count up front",
            "while", "Loop until a condition fails",
        )
        self.reveal(compare)
        self.narrate("When you know how many passes you need, reach for a for loop. When you do not, a while loop reads more naturally.")
        self.dismiss(compare)

        # --- Recap ------------------------------------------------------------
        recap = Recap([
            "A for loop has four parts in a fixed order",
            "The condition is checked BEFORE each pass",
            "Forget the step and you loop forever",
        ])
        self.reveal(recap)
        self.narrate(
            "To recap: a for loop has four parts in a fixed order, the condition is always "
            "checked first, and forgetting the step gives you an infinite loop."
        )
