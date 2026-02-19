# PART 5

## CLI COLORS, VISUAL HIERARCHY, AND UX RULES

---

## 41. CLI Color Philosophy

Colors are used to **improve scan readability**, not decoration.

Rules:

- Colors are **CLI-only**
- Colors never affect exported data
- Colors must degrade gracefully
- Colors must be disable-able
- Output must remain readable in plain text

---

## 42. Color Control Flags

### New flags

| Flag        | Description           |
| ----------- | --------------------- |
| `-color`    | Force color output    |
| `-no-color` | Disable color output  |
| `-mono`     | Alias for `-no-color` |

### Default behavior

- Enable colors if:
  - Stdout is a TTY

- Disable colors if:
  - Output is piped
  - `-no-color` is set

This follows standard Unix behavior.

---

## 43. Color Palette (Final)

Use **ANSI colors only**. No truecolor.

| Element          | Color      | Reason              |
| ---------------- | ---------- | ------------------- |
| URL              | Cyan       | Primary scan target |
| Domain           | Cyan       | Consistency         |
| Technology name  | Green      | Positive detection  |
| Headers label    | Blue       | Context grouping    |
| Body label       | Blue       | Context grouping    |
| Progress numbers | Yellow     | Visibility          |
| Warnings         | Yellow     | Non-fatal           |
| Errors           | Red        | Fatal / blocking    |
| Info text        | Dim / Gray | Noise reduction     |

---

## 44. Semantic Color Meanings

Colors always mean the same thing.

- **Green** = detected technology
- **Cyan** = target (URL or domain)
- **Blue** = section header
- **Yellow** = attention needed
- **Red** = failure

Never reuse colors for different meanings.

---

## 45. CLI Output (All Mode, Colored)

Example (conceptual):

```
URL: https://example.com        ← cyan

Headers:                        ← blue
  nginx                         ← green
  Cloudflare                    ← green

Body:                           ← blue
  React                         ← green
```

If color is disabled, output becomes:

```
URL: https://example.com

Headers:
  nginx
  Cloudflare

Body:
  React
```

---

## 46. CLI Output (Domain Mode, Colored)

```
Domain: example.com             ← cyan

Technologies:                   ← blue
  nginx                         ← green
  Cloudflare                    ← green
  React                         ← green
```

---

## 47. Progress Output (Colored)

Progress should be **compact and visible**.

```
[+] Total: 200                  ← yellow numbers
[+] Completed: 83
[+] Remaining: 117
```

Text symbols stay neutral. Only numbers are colored.

---

## 48. Warning and Error Styling

### Warning (non-fatal)

```
[!] timeout on https://example.com   ← yellow
```

### Error (fatal)

```
[-] wappalyzer fingerprints not found   ← red
```

Warnings never exit the program.
Errors always exit non-zero.

---

## 49. Confidence-Based Color (Optional Enhancement)

If confidence levels are shown in CLI:

| Confidence | Color  |
| ---------- | ------ |
| high       | Green  |
| medium     | Yellow |
| low        | Red    |

Example:

```
React (high)
jQuery (medium)
```

This is **CLI-only** and optional.

---

## 50. Color Implementation Strategy (Go)

### Centralized color handling

Create a single utility:

```
util/color.go
```

Responsibilities:

- Detect TTY
- Enable / disable colors
- Provide named color functions

Example API:

```go
Color.Cyan("text")
Color.Green("text")
Color.Blue("text")
Color.Yellow("text")
Color.Red("text")
Color.Dim("text")
```

No raw ANSI codes outside this file.

---

## 51. Writer Rules (Important)

- CSV writer: **never color**
- JSON writer: **never color**
- TXT writer: **never color**
- MD writer: **never color**
- CLI writer: **may color**

If `-o` is set and output is redirected, CLI output should auto-disable colors.

---

## 52. Accessibility Rules

- Do not rely on color alone
- Labels and symbols must still communicate meaning
- Colors enhance, not replace, structure

Example:

```
Headers:
Body:
```

Never removed, even when colored.

---

## 53. Performance Impact

- Color formatting is string-level only
- No measurable performance impact
- Disabled automatically in high-volume piping

---

## 54. Final CLI Color Guarantees

This design ensures:

- Clean, readable output during recon
- Script-safe behavior
- No accidental color bleed into exports
- Consistent semantics
- Easy future theming
