# Cases for `gomd -r` and the TUI

Prose mentions x before the span `x`, and the word code before `code`.

A long paragraph that keeps going so that the inline span lands near the wrap point `wrapped span here` and continues after it with more words.

A link whose text wraps across the edge of the pane [this link text is long enough to wrap](https://example.com/wrap) at narrow widths.

## Wrapped code

```text
short line
a very long line in a code block that is much wider than the content pane so glamour has to wrap it onto more lines
end
```

```text
the block after the wrapped one
```

## Blank lines and empty blocks

```go
func a() {

	return
}
```

```
```

After the empty block.

## Nested blocks

- A list item with code:

  ```sh
  echo in-list
  ```

- Second item with `inline` code.

> A quote with code:
>
> ```sh
> echo in-quote
> ```

## Table spans

| Key | Action |
| --- | ------ |
| `j` | down `1` |
| `k` | up `1` |

## Indented code

    indented line one
    indented line two

The end.
