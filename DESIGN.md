---
name: SSH Drop
description: Terminal-native file handoff with a visible transfer route.
colors:
  lime-dark: "#B4F566"
  lime-light: "#356300"
  ink-dark: "#EAF0EA"
  ink-light: "#182522"
  muted-dark: "#A6BAB4"
  muted-light: "#52645C"
  error-dark: "#FF8C96"
  error-light: "#A32633"
  warning-dark: "#EAB95A"
  warning-light: "#805000"
typography:
  title:
    fontWeight: 700
  selected:
    fontWeight: 700
components:
  title-dark:
    textColor: "{colors.ink-dark}"
    typography: "{typography.title}"
  title-light:
    textColor: "{colors.ink-light}"
    typography: "{typography.title}"
  selected-dark:
    textColor: "{colors.lime-dark}"
    typography: "{typography.selected}"
  selected-light:
    textColor: "{colors.lime-light}"
    typography: "{typography.selected}"
  muted-dark:
    textColor: "{colors.muted-dark}"
  muted-light:
    textColor: "{colors.muted-light}"
  error-dark:
    textColor: "{colors.error-dark}"
  error-light:
    textColor: "{colors.error-light}"
  warning-dark:
    textColor: "{colors.warning-dark}"
  warning-light:
    textColor: "{colors.warning-light}"
---

# Design System: SSH Drop

## Overview

**Creative North Star: "Route Line"**

Route Line makes the file handoff legible through one input and a visible FILE → SSH → CLIPBOARD route. Terminal monospace, sparse spacing, restrained borders and lime focus keep repeated drops quick. The terminal owns the font and background.

**Key Characteristics:**

- One dominant path input
- Persistent remote context
- Text and symbols alongside semantic color
- Compact layouts measured in terminal cells

## Colors

Adaptive foregrounds support both light and dark terminal themes. Frontmatter suffixes identify the terminal background mode; they are not background tokens.

### Primary

- **Lime:** brand, selected remote, focused field border, active route stage and successful outcomes.

### Secondary

- **Amber:** clipboard warnings, cancellation and quit confirmation.
- **Rose red:** invalid input, authentication rejection and transfer errors.

### Neutral

- **Ink:** headings and entered text.
- **Muted sage:** instructions, remote metadata, inactive route stages and keyboard help.

**The Outcome Rule.** Color reinforces written outcomes; it never carries status alone.

## Typography

Terminal-owned monospace at the user's configured size. Bold marks titles and selected items; body, input and metadata use regular weight. No application font family, size or letter spacing.

## Layout

Left-aligned flow, capped at 84 columns. Outer inset: 1 row vertically, 2 columns horizontally. Inner width: `max(1, min(84, terminal width) − 4)`. Standard screens separate groups with blank rows.

Below 20 rows, collapse vertical gaps and supporting copy; the path input loses its box, while route and keyboard actions remain. Below 16 rows, the password panel drops the brand and centering. Below 40 inner columns, CLIPBOARD becomes COPY. Long content wraps by terminal cell width; limited detail areas truncate with an ellipsis.

## Elevation & Depth

Flat terminal canvas, no shadows or application background fills. Password entry is centered through spacing when height permits; its boundary alone distinguishes it.

## Shapes

Rounded terminal box-drawing borders enclose the standard path field and password panel. Borders use lime, with 1 column of horizontal internal padding. Route connectors are horizontal line glyphs; selection uses `>`.

## Components

### Path input

Prominent, focused, lime prompt and border; regular ink text and muted placeholder. Standard input text width: `max(1, inner width − 7)`. Compact text width: `max(1, inner width − 3)`. Blur during upload. Clear after transfer success and invite another drop.

### Route

FILE → SSH → CLIPBOARD. Active stages use lime, inactive stages muted text; completed stages add `✓`, upload adds `…`, clipboard failure adds amber `!`. Connectors stretch to available inner width.

### Remote picker and context

Bold ink rows; selected row adds lime and `>`. Destination appears beneath each remote. Show a position indicator for a scrollable list. Remote identity remains visible during connection and repeated uploads.

### Password panel

Lime border, ink heading, muted host and actions. Mask with `*`; clear submitted passwords. Text input caps at 28 columns; panel caps at 46 columns, both shrinking to available space. Red rejection text stays inside the panel.

### Result and keyboard help

Destination precedes source after transfer. Keep success, clipboard warning, error and cancellation messages distinct. Muted footer shows state-specific commands; amber confirmation asks before quitting an active upload. Commands `r` and `q` act as shortcuts only when the path input is empty.

## Do's and Don'ts

### Do:

- **Do** pair outcome colors with explicit text and route symbols.
- **Do** keep the selected remote visible across repeated drops.
- **Do** preserve destination paths after successful uploads.
- **Do** measure wrapping and truncation in terminal cells.

### Don't:

- **Don't** assign a font, pixel scale or background owned by the terminal.
- **Don't** report clipboard failure as transfer failure.
- **Don't** use decorative panels or animations that distract from the file action.
