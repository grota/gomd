# test

```
some block
some block
some block
```

```bash
echo "some code"
if [ -f "test-images/gradient.png" ]; then
  echo "image exists"
fi
```

## Images (TUI and `gomd -r`)

Each case says what you should see.

### 1. Standalone image

The caption line is replaced by the gradient image.

![gradient](./test-images/gradient.png)

### 2. Inline image inside a sentence (B2 fix)

This text must stay: before ![red square](test-images/red.png) after. The red square appears on the line below it.

### 3. HTML img tag

<img src="test-images/checker.png" alt="checker">

### 4. Empty alt text

![](test-images/checker.png)

### 5. Same image used twice

![red one](test-images/red.png)

![red two](test-images/red.png)

### 6. Path with an encoded space

![blue](test-images/sub%20dir/blue.png)

### 7. Image inside a list item

- first item
- ![gradient in list](test-images/gradient.png)
- third item

### 8. Missing file

Stays as text, no image: ![missing](test-images/nope.png)

### 9. Remote URL

Stays as text, not fetched: ![remote](https://example.com/logo.png)

### 10. Image with a title

![titled](test-images/checker.png "A checkerboard")
