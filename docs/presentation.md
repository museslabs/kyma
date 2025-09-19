---
title: Welcome
image_backend: docs
style:
  border: hidden
---

![img|43x10](kyma_logo.png)

> Κῦμα (Kyma) - Ancient Greek: A wave, billow, or surge; metaphorically
representing the flow and movement of ideas and presentations.

A terminal-based presentation tool that creates beautiful presentations from
markdown files with smooth animated transitions.

## Basic navigation:

- **Next slide**: `→`, `l`, or `Space`
- **Previous slide**: `←` or `h`
- **Quit**: `q`, `Esc`, or `Ctrl+C`

----
---
title: Features
transition: swipeLeft
style:
  border: hidden
  theme: dracula
---

# Features

- **Markdown-based**: Write your presentations in simple markdown syntax
- **Rich rendering**: Beautiful terminal rendering using Glamour markdown renderer
- **Smooth transitions**: Multiple animated transition effects between slides
- **Image Rendering**: Render high quality images
- **Hot reload**: Live reloading of presentation files during editing by default
- **Customizable styling**: Configure borders, colors, and layouts via YAML
- **Theme support**: Choose from built-in Glamour themes or load custom JSON theme
- **Flexible layouts**: Center, align, and position content with various layouts
- **Simple navigation**: Intuitive keyboard controls for presentation flow
  - Command palette with slide search and filtering
  - Direct slide jumping by number
  - Multi-slide forward/backward jumping
  - Quick first/last slide navigation
- **Presentation timer**: Built-in timer system with per-slide and global timing
  - Toggle timer display with a single key
  - Track time spent on each slide
  - Monitor total presentation duration
  - Automatic pause/resume during slide transitions

----
---
title: Transitions
transition: swipeLeft
style:
  border: hidden
  theme: dracula
---

# Available transitions

- `none` - No transition (default)
- `swipeLeft` - Slide swipes in from right to left
- `swipeRight` - Slide swipes in from left to right
- `slideUp` - Slide slides up from bottom
- `slideDown` - Slide slides down from top
- `flip` - Flip transition effect

----
---
title: Styles
transition: slideUp
style:
  border: rounded
  border_color: "#FF0000"
  layout: center
  theme: tokyo-night
---

# Styling and theme support

- `border` - normal, rounded, double, thick, hidden, block
- `border_color` - Hex color for border (or "default" for theme-based color)
- `layout` - center, left, right, top, bottom
- `theme` - predefined theme name or path to custom JSON theme file

----
---
title: Style usage
style:
  border: hidden
transition: swipeLeft
---

# Style usage

To use these styles you can do so by writing yaml at the top of each slide in
wrapped between three dashes `---`

```yaml
transition: swipeLeft
style:
  border: rounded
  border_color: "#FF0000"
  layout: center
  theme: dracula
```

----
---
title: Config
preset: dark
---

# Configuration

A configuration file is created by default in `~/.config/kyma.yaml` but you
can use another config file using the `-c` flag or by having a `kyma.yaml`
file present in the directory you are executing the command from.

A `kyma.yaml` file looks like this:

```yaml
global:
  style:
    border: rounded
    border_color: "#9999CC"
    layout: center
    theme: dracula

presets:
  minimal:
    style:
      border: hidden
      theme: notty
  dark:
    style:
      border: rounded
      theme: dracula
```

----
---
title: Global styles
---

# Global styles

You can define a global style config that all slides use if no configuration is
provided.

If we take a look at our `kyma.yaml` from before you can see the global
configuration in lines 1-6

```yaml{1-6} --numbered
global:
  style:
    border: rounded
    border_color: "#9999CC"
    layout: center
    theme: dracula

presets:
  minimal:
    style:
      border: hidden
      theme: notty
  dark:
    style:
      border: rounded
      theme: dracula
```

----
---
title: Presets
---

# Presets

Presets are a way to define reusable style configurations to apply to individual
slides withou having to copy pase each time

If we take a look again at our `kyma.yaml` from before you can see the presets
in lines 8-16

```yaml{8-16} --numbered
global:
  style:
    border: rounded
    border_color: "#9999CC"
    layout: center
    theme: dracula

presets:
  minimal:
    style:
      border: hidden
      theme: notty
  dark:
    style:
      border: rounded
      theme: dracula
```

----
---
title: More ways to navigate
---

# More ways to navigate

- **Last slide**: `End`, `Shift+↓`, or `$`
- **Command palette**: `/` or `p` - Opens a searchable list of all slides for quick navigation
- **Go to slide**: `g` or `:` - Jump directly to a specific slide number
- **Jump slides**: `1-9` + `h`/`←` or `l`/`→` - Jump multiple slides backward/forward (e.g., `5h` jumps 5 slides back)

----
---
title: Grid layouts
transition: swipeLeft
image_backend: docs
---

# Grid Layout

You can create grid layouts by using the `[grid]` and `[column]` tags with their
respective closing tags that start with a backslash `/`

[grid]
[column]
```go
package main

import "fmt"

func main() {
  fmt.Println("Hello World")
}
```
[/column]
[column]
```c
#include <stdio.h>

int main(void) {
  printf("Hello World\n");
  return 0;
}
```
[/column]
[column]
```rust
fn main() {
  println!("Hello World");
}
```
[/column]
[/grid]

----
---
title: Achievements
transition: swipeLeft
image_backend: docs
---

# Achievements

_Kyma being used for a talk at Laravel Greece's 10 year anniversary meetup in Athens_

![img|60x18](laratalk.jpg)

