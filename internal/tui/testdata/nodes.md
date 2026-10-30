# Nodes

Every syntax that gives a selectable node, and syntax that looks like one
but does not.

## Inline links

A [plain link](https://example.com/plain) and one with a
[title](https://example.com/title "The title") and an
[angle destination](<https://example.com/with space>).

A URL with [parentheses](https://en.wikipedia.org/wiki/Magic_number_(programming)#Unnamed)
and a [space before it]( https://example.com/space).

A [link whose text
goes on](https://example.com/two-lines) and one whose destination does: [here](
https://example.com/next-line).

Nested brackets: [[Become a backer](https://example.com/backer)].

A [link with `code`](https://example.com/code) and [**bold**](https://example.com/bold).

An [anchor](#inline-links) to a heading.

## Reference links

The [full][ref], the [collapsed][] and the [shortcut] forms, and an image
![badge][img].

[ref]: https://example.com/ref
[collapsed]: https://example.com/collapsed "Collapsed"
[shortcut]: <https://example.com/shortcut>
[img]: https://example.com/badge.svg

## Autolinks and bare URLs

An autolink <https://example.com/auto>, a bare https://example.com/bare.
Ending a sentence: https://example.com/end, or (https://example.com/paren).
Mail <someone@example.com> and bare someone@example.com.

Side by side: [one](https://example.com/1) [two](https://example.com/2),
[same](https://example.com/1) [url](https://example.com/1), [anchor](#images)
[then](https://example.com/3), and [a link](https://example.com/4) <https://example.com/5>.

## Images

An ![inline image](img/a.png) and a linked one:
[![badge](https://example.com/b.svg)](https://example.com/target).

## HTML

Inline <a href="https://example.com/a-tag">anchor tag</a> and
<img src="img/tag.png" alt="image tag"> in a paragraph.

<p align="center">
  <a href="https://example.com/block-a"><img src="img/logo.png" alt="Logo"></a>
  Text with `code` and [markdown](https://example.com/md) in an HTML block.
</p>

## Code spans

A `span`, a `` span with ` inside ``, and `` ` `` a backtick alone.

A span that `goes on
the next line` here.

| Operator | Meaning |
| -------- | ------- |
| `a \| b` | either  |
| `[x](y)` | no link |

Not a link inside code: `https://example.com/in-code` and `[t](u)`.

### Heading with `code`

## Code blocks

```go
fmt.Println("fenced")
```

~~~
tilde fence
~~~

```sh
trailing blank line follows

```

- A list item:

  ```sh
  echo in-list
  ```

> Quoted:
>
> ```
> in quote
> ```

    indented code
    and its second line

```
```

Unclosed at the end:

```text
never closed
