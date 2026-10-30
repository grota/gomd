---
title: Edge cases
tags: [snapshot]
---

# Edge cases

Inline `code`, `two words`, ``with ` tick``, and `j`/`k` keys. A **bold
`span`** and *emphasis with a [link](https://example.com/a "Title")*.

Bare URL https://example.com/path, autolink <https://example.com/auto> and
mail <someone@example.com>. Unicode: café, 日本語, emoji 🎉 `ünïcode`.

## Links and anchors

- [same file](#links-and-anchors)
- [repeated](#usage-1)
- [other file](docs/other%20file.md#setup)
- <a href="https://example.com/html">html link</a>

## Usage

First usage.

## Usage

Second usage.

## Code

````md
```go
fmt.Println("nested")
```
````

~~~
tilde fence
~~~

```html
<img src="a.png" alt="x"> and <a href="u">t</a>
```

    indented code block

> A quote with `code` and a [link](https://example.com/q).

| Column | Value |
| ------ | ----- |
| `a`    | [b](https://example.com/b) |
| 日本   | 🎉    |

1. one
2. two `2`
   - nested *item*

---

Text after a rule, then a final paragraph.
