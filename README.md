MA-FSA for Go
=============

Package mafsa implements Minimal Acyclic Finite State Automata (MA-FSA)
with Minimal Perfect Hashing (MPH). Basically, it's a set of strings that
lets you test for membership, do spelling correction (fuzzy matching)
and autocomplete, but with higher memory efficiency than a regular
trie. With MPH, you can associate each entry in the tree with data from
your own application.

In this package, MA-FSA is implemented by two types:

- BuildTree
- MinTree

A BuildTree is used to build data from scratch. Once all the elements
have been inserted, the BuildTree can be serialized into a byte slice
or written to a file directly. It can then be decoded into a MinTree,
which uses significantly less memory. MinTrees are read-only, but this
greatly improves space efficiency.


## Tutorial

Create a BuildTree and insert your items in lexicographical order. Be
sure to call `Finish()` when you're done.

```go
bt := mafsa.New()
bt.Insert("cities") // an error will be returned if input is < last input
bt.Insert("city")
bt.Insert("pities")
bt.Insert("pity")
bt.Finish()
```

The tree is now compressed to a minimum number of nodes and is ready to
be saved.

```go
err := bt.Save("filename")
if err != nil {
    log.Fatal("Could not save data to file:", err)
}
```

In your production application, then, you can read the file into a
MinTree directly:

```go
mt, err := mafsa.Load("filename")
if err != nil {
    log.Fatal("Could not load data from file:", err)
}
```

The `mt` variable is a `*MinTree` which has the same data as the original
BuildTree, but without all the extra "scaffolding" that was required
for adding new elements. In our testing, we were able to store over 8
million phrases (average length 24, much longer than words in a typical
dictionary) in as little as 2 GB on a 64-bit system.

The package provides some basic read mechanisms.

```go
// See if a string is a member of the set
fmt.Println("Does tree contain 'cities'?", mt.Contains("cities"))
fmt.Println("Does tree contain 'pitiful'?", mt.Contains("pitiful"))

// You can traverse down to a certain node, if it exists
fmt.Printf("'y' node is at: %p\n", mt.Traverse([]rune("city")))

// To traverse the tree and get the number of elements inserted
// before the prefix specified
node, idx := mt.IndexedTraverse([]rune("pit"))
fmt.Println("Index number for 'pit': %d", idx)
```

To associate entries in the tree with data from your own application,
create a slice with the data in the same order as the elements were
inserted into the tree:

```go
myData := []string{
    "The plural of city",
    "Noun; a large town",
    "The state of pitying",
    "A feeling of sorrow and compassion",
}
```

The index number returned with `IndexedTraverse()` (usually minus 1)
will get you to the element in your slice if you traverse directly to
a final node:

```go
node, i := mt.IndexedTraverse([]rune("pities"))
if node != nil && node.Final {
    fmt.Println(myData[i-1])
}
```

If you do `IndexedTraverse()` directly to a word in the tree, you must
-1 because that method returns the number of elements in the tree before
those that *start* with the specified prefix, which is non-inclusive
with the node the method landed on.

There are many ways to apply MA-FSA with minimal perfect hashing, so
the package only provides the basic utilities. Along with `Traverse()`
and `IndexedTraverse()`, the edges of each node are exported so you may
conduct your own traversals according to your needs.


## Encoding Format

This section describes the file format used by `Encoder` and
`Decoder`. You probably will never need to implement this yourself;
it's already done for you in this package.

BuildTrees can be encoded as bytes and then stored on disk or decoded
into a MinTree. The encoding of a BuildTree is a binary file that is
composed of a sequence of words, which is a sequence of big-endian
bytes. The file is inspected one word at a time.

The word length is equal to (in bytes): 1 (the flag byte) + character
length (UTF-8 encoding of the unicode point) + pointer length (byte 1
in the header).

The first word contains file format information. Byte 0 is the file
version (right now, 2). Byte 1 is the pointer length. In the first word
specifying the file format, we assume a character length of 1 and fill
up the rest of the bytes with zeros bytes.

Package Shugyousha/mafsa initializes the file with this first word entry
by default:

    []byte{0x02 0x04 0x00 0x00 0x00 0x00}

This indicates that we are using version 2 of the file format and each
pointer to another node is 4 bytes. This pointer size allows us to encode
trees with up to 715.8 Mio. edges. The rest of the first entry are filled
with zero bytes until the minimal size of a word has been reached (the
pointer size takes the position of the UTF-8-character in a regular word).

Every other word in the file represents an edge (not a node). Those
words consist of bytes according to this format:

    [flag byte] [UTF-8-encoded character] [pointer]

The length of the character is encoded in the bits 2-4 (zero-indexed
from the least significant one) of the flags, the pointer size in bytes
is specified in the second byte of the initial word, and the flags part
is always a single byte. This allows us to have 8 flags per edge, but
currently only 5 are used.

The flag bits are:

- `0x01` = End of Word (EOW, or final)
- `0x02` = End of Node (EON)
- `0x04` = lsb of character length (three bits in total)
- `0x08` = middle bit of character length (three bits in total)
- `0x10` = msb of character length (three bits in total)

A node essentially consists of a consecutive, variable-length sequence
of words, where each word represents an edge. To encode a node, write
each edge sequentially. Set the final (EOW) flag if the node it points
to is a final node, and set the EON flag if it is the last edge for that
node. The EON flag indicates the next word is an edge belonging to a
different node. The pointer in each word should be the *byte* index of
the start of the node being pointed to.

For example, the tree containing these words:

```
-- dog
-- dogs
-- hello
-- jello
-- été
-- あello
```

Encodes to (each line is one word, note that they are varying in size
depending on the UTF-8 encoding length of the character):

```
 1   02 04 00 00 00 00
 2   04 64 00 00 00 27
 3   04 68 00 00 00 2d
 4   04 6a 00 00 00 33
 5   08 c3 a9 00 00 00 39
 6   0e e3 81 82 00 00 00 3f
 7   06 6f 00 00 00 45
 8   06 65 00 00 00 4b
 9   06 65 00 00 00 4b
10   06 74 00 00 00 51
11   06 65 00 00 00 4b
12   07 67 00 00 00 58
13   06 6c 00 00 00 5e
14   0b c3 a9 00 00 00 00
15   07 73 00 00 00 00
16   06 6c 00 00 00 64
17   07 6f 00 00 00 00
```

Or here's the hexdump (this makes it easier to follow the byte offsets):

```
00000000: 0204 0000 0000 0464 0000 0027 0468 0000  .......d...'.h..
00000010: 002d 046a 0000 0033 08c3 a900 0000 390e  .-.j...3......9.
00000020: e381 8200 0000 3f06 6f00 0000 4506 6500  ......?.o...E.e.
00000030: 0000 4b06 6500 0000 4b06 7400 0000 5106  ..K.e...K.t...Q.
00000040: 6500 0000 4b07 6700 0000 5806 6c00 0000  e...K.g...X.l...
00000050: 5e0b c3a9 0000 0000 0773 0000 0000 066c  ^........s.....l
00000060: 0000 0064 076f 0000 0000                 ...d.o....
```

Now, this tree isn't a thorough-enough example to test your implementation
against, but it should illustrate the idea. First notice that the first
word is the initializer word. The second word is the first edge coming
off the root node, and words 3 and 4 are both edges coming off the root
node. The word at 6 has the EON flag set (0x02) which indicates the end
of the edges coming off that node (the root node). You'll see that edges
at word indices 8 and 9 both point to the node starting at edge with word
index 16 (at byte offset 0x4b, 75 in decimal). That would be the shared
'l' node after 'he' and 'je'.

If a node has no outgoing edges, the pointer bits are 0.
